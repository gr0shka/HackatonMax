package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"HackatonMax/config"
	"HackatonMax/internal/clients/ml"
	"HackatonMax/internal/clients/places"
	"HackatonMax/internal/clients/routing"
	"HackatonMax/internal/repository/postgres"
	"HackatonMax/internal/service"
	"HackatonMax/internal/transport/rest"
	v1 "HackatonMax/internal/transport/rest/v1"
)

// Module wires all application dependencies into an Uber Fx application.
var Module = fx.Options(
	fx.Provide(
		config.LoadConfig,
		newLogger,
		newPostgresPool,
		newRepositories,
		newClients,
		newServices,
		newHandlers,
		newRouter,
	),
	fx.Invoke(startHTTPServer),
)

func newLogger(cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func newPostgresPool(lc fx.Lifecycle, cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := postgres.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres pool: %w", err)
	}

	if cfg.Postgres.AutoMigrate {
		logger.Info("running database migrations", slog.String("dir", cfg.Postgres.MigrationsDir))
		if err := postgres.RunMigrations(ctx, pool, cfg.Postgres.MigrationsDir); err != nil {
			postgres.ClosePostgresPool(pool)
			return nil, fmt.Errorf("database auto-migration failed: %w", err)
		}
		logger.Info("database migrations applied successfully")
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing postgres connection pool")
			postgres.ClosePostgresPool(pool)
			return nil
		},
	})

	return pool, nil
}

type Repositories struct {
	fx.Out

	UserRepo  service.UserRepository
	PlaceRepo service.PlaceRepository
}

func newRepositories(pool *pgxpool.Pool) Repositories {
	return Repositories{
		UserRepo:  postgres.NewUserRepository(pool),
		PlaceRepo: postgres.NewPlaceRepository(pool),
	}
}

type Clients struct {
	fx.Out

	PlacesProvider service.PlacesProvider
	MLClient       service.MLClient
	RoutingClient  service.RoutingClient
}

func newClients(cfg *config.Config, placeRepo service.PlaceRepository, logger *slog.Logger) Clients {
	twoGis := places.NewTwoGisClient(places.TwoGisConfig{
		BaseURL:     cfg.TwoGIS.BaseURL,
		APIKey:      cfg.TwoGIS.APIKey,
		RPS:         cfg.TwoGIS.RPS,
		Burst:       cfg.TwoGIS.Burst,
		HTTPTimeout: cfg.TwoGIS.HTTPTimeout,
		CacheTTL:    cfg.TwoGIS.CacheTTL,
		CacheSize:   cfg.TwoGIS.CacheSize,
	})

	fallbackPlaces := places.NewFallbackPlacesProvider(twoGis, placeRepo, logger)

	mlClient := ml.NewClient(ml.Config{
		BaseURL: cfg.ML.BaseURL,
		Timeout: cfg.ML.Timeout,
	})

	routingClient := routing.NewOSRMClient(routing.OSRMConfig{
		BaseURL:     cfg.OSRM.BaseURL,
		HTTPTimeout: cfg.OSRM.Timeout,
	})

	return Clients{
		PlacesProvider: fallbackPlaces,
		MLClient:       mlClient,
		RoutingClient:  routingClient,
	}
}

type Services struct {
	fx.Out

	UserService  service.UserService
	RouteService service.RouteService
}

func newServices(
	userRepo service.UserRepository,
	placesProvider service.PlacesProvider,
	mlClient service.MLClient,
	routingClient service.RoutingClient,
) Services {
	return Services{
		UserService:  service.NewUserService(userRepo),
		RouteService: service.NewRouteService(userRepo, placesProvider, mlClient, routingClient),
	}
}

func newHandlers(routeSvc service.RouteService, userSvc service.UserService) *v1.Handler {
	return v1.NewHandler(routeSvc, userSvc)
}

func newRouter(handler *v1.Handler, cfg *config.Config) *chi.Mux {
	return rest.NewRouter(handler, rest.RouterConfig{
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
	})
}

func startHTTPServer(lc fx.Lifecycle, router *chi.Mux, cfg *config.Config, logger *slog.Logger) {
	addr := net.JoinHostPort(cfg.HTTP.Host, cfg.HTTP.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting HTTP server", slog.String("addr", addr))
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("failed to listen on %s: %w", addr, err)
			}
			go func() {
				if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server stopped unexpectedly", slog.Any("error", err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down HTTP server gracefully")
			return server.Shutdown(ctx)
		},
	})
}
