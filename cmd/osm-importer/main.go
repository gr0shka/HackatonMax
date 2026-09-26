package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"HackatonMax/internal/importer/osm"
)

func main() {
	// Attempt loading .env if available
	_ = godotenv.Load()

	// 1. Resolve defaults from environment variables
	defaultCity := os.Getenv("CITY")
	if defaultCity == "" {
		defaultCity = osm.DefaultCity
	}

	defaultLimit := osm.DefaultLimit
	if limitStr := os.Getenv("LIMIT"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			defaultLimit = parsed
		}
	}

	defaultDBURL := os.Getenv("DATABASE_URL")
	if defaultDBURL == "" {
		defaultDBURL = "postgres://postgres:postgres@localhost:5432/route_optimizer?sslmode=disable"
	}

	defaultBBox := os.Getenv("BBOX")
	defaultOverpassURL := os.Getenv("OVERPASS_URL")
	if defaultOverpassURL == "" {
		defaultOverpassURL = osm.DefaultOverpassURL
	}

	// 2. Parse command-line flags
	var (
		city        string
		limit       int
		dbURL       string
		bbox        string
		overpassURL string
		timeoutSec  int
		batchSize   int
	)

	flag.StringVar(&city, "city", defaultCity, "Target city name (e.g. Moscow, SPb) or bounding box 'south,west,north,east'")
	flag.IntVar(&limit, "limit", defaultLimit, "Maximum number of places to import (default: 500)")
	flag.StringVar(&dbURL, "db", defaultDBURL, "PostgreSQL database connection URL (or DATABASE_URL env)")
	flag.StringVar(&bbox, "bbox", defaultBBox, "Direct bounding box coordinates override 'south,west,north,east'")
	flag.StringVar(&overpassURL, "overpass-url", defaultOverpassURL, "Overpass API interpreter endpoint URL")
	flag.IntVar(&timeoutSec, "timeout", 30, "HTTP timeout in seconds for Overpass API requests (default: 30s)")
	flag.IntVar(&batchSize, "batch-size", 100, "Database batch insert chunk size (default: 100)")
	flag.Parse()

	// 3. Setup structured logger
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)

	logger.Info("==================================================")
	logger.Info("       OSM POI SEEDER / IMPORTER INTO POSTGIS     ")
	logger.Info("==================================================")
	logger.Info("Configuration loaded",
		slog.String("city", city),
		slog.Int("limit", limit),
		slog.String("bbox", bbox),
		slog.String("overpass_url", overpassURL),
		slog.Int("timeout_sec", timeoutSec),
		slog.Int("batch_size", batchSize),
	)

	// 4. Setup graceful context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 5. Connect to PostgreSQL
	logger.Info("Connecting to PostgreSQL database...")
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()

	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Invalid database URL", slog.Any("error", err))
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(dbCtx, poolCfg)
	if err != nil {
		logger.Error("Failed to create connection pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(dbCtx); err != nil {
		logger.Error("Database connection ping failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Database connection established successfully")

	// 6. Initialize Importer
	impCfg := osm.Config{
		OverpassURL: overpassURL,
		City:        city,
		BBox:        bbox,
		Limit:       limit,
		HTTPTimeout: time.Duration(timeoutSec) * time.Second,
		BatchSize:   batchSize,
	}

	importer := osm.NewImporter(impCfg, pool, logger)

	// 7. Execute POI seeding
	stats, err := importer.Run(ctx)
	if err != nil {
		logger.Error("OSM Seeding failed", slog.Any("error", err))
		os.Exit(1)
	}

	// 8. Output final statistics
	fmt.Println()
	logger.Info("==================================================")
	logger.Info("           OSM SEEDING SUMMARY REPORT             ")
	logger.Info("==================================================")
	logger.Info("Seeding completed successfully",
		slog.String("city", stats.City),
		slog.String("bbox", stats.BBox),
		slog.Int("fetched_from_osm", stats.FetchedCount),
		slog.Int("filtered_without_name_or_coords", stats.FilteredCount),
		slog.Int("valid_places_ready", stats.ParsedCount),
		slog.Int("newly_inserted_into_db", stats.InsertedCount),
		slog.Int("skipped_existing_duplicates", stats.SkippedCount),
		slog.Duration("total_duration", stats.Duration),
	)
	logger.Info("==================================================")
}
