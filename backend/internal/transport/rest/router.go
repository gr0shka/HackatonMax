package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "HackatonMax/docs"
	v1 "HackatonMax/internal/transport/rest/v1"
	"HackatonMax/internal/transport/rest/v1/dto"
)

// RouterConfig holds options for HTTP routing.
type RouterConfig struct {
	AllowedOrigins []string
}

// NewRouter sets up Chi router with middleware, Swagger UI, and v1 endpoints.
func NewRouter(v1Handler *v1.Handler, cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS configuration
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check endpoint
	r.Get("/health", HealthCheck)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", HealthCheck)

		r.Route("/users", func(r chi.Router) {
			r.Post("/", v1Handler.CreateOrUpdateUser)
			r.Get("/{id}", v1Handler.GetUser)
		})

		r.Route("/routes", func(r chi.Router) {
			r.Post("/build", v1Handler.BuildRoute)
		})
	})

	return r
}

// HealthCheck godoc
// @Summary      Проверка состояния сервиса
// @Description  Проверка статуса сервиса и подключения к БД.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.HealthResponse "Сервис функционирует нормально"
// @Router       /health [get]
// @Router       /api/v1/health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto.HealthResponse{Status: "ok"})
}
