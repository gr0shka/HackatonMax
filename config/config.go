package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config aggregates all application configuration values.
type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Postgres PostgresConfig `yaml:"postgres"`
	TwoGIS   TwoGISConfig   `yaml:"twogis"`
	ML       MLConfig       `yaml:"ml"`
	OSRM     OSRMConfig     `yaml:"osrm"`
	Log      LogConfig      `yaml:"log"`
}

// HTTPConfig specifies HTTP server parameters.
type HTTPConfig struct {
	Port           string        `env:"HTTP_PORT" env-default:"8080"`
	Host           string        `env:"HTTP_HOST" env-default:"0.0.0.0"`
	ReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	AllowedOrigins []string      `env:"HTTP_ALLOWED_ORIGINS" env-default:"*"`
}

// PostgresConfig specifies database connection parameters.
type PostgresConfig struct {
	DSN            string `env:"DATABASE_URL" env-default:"postgres://postgres:postgres@localhost:5432/route_optimizer?sslmode=disable"`
	MigrationsDir  string `env:"MIGRATIONS_DIR" env-default:"./migrations"`
	AutoMigrate    bool   `env:"AUTO_MIGRATE" env-default:"true"`
}

// TwoGISConfig specifies 2GIS Places API settings.
type TwoGISConfig struct {
	BaseURL     string        `env:"TWOGIS_BASE_URL" env-default:"https://catalog.api.2gis.com"`
	APIKey      string        `env:"TWOGIS_API_KEY" env-default:"demo-key"`
	RPS         float64       `env:"TWOGIS_RPS" env-default:"5.0"`
	Burst       int           `env:"TWOGIS_BURST" env-default:"5"`
	HTTPTimeout time.Duration `env:"TWOGIS_TIMEOUT" env-default:"5s"`
	CacheTTL    time.Duration `env:"TWOGIS_CACHE_TTL" env-default:"15m"`
	CacheSize   int           `env:"TWOGIS_CACHE_SIZE" env-default:"1000"`
}

// MLConfig specifies ML ranking service settings.
type MLConfig struct {
	BaseURL string        `env:"ML_BASE_URL" env-default:"http://localhost:8000"`
	Timeout time.Duration `env:"ML_TIMEOUT" env-default:"5s"`
}

// OSRMConfig specifies OSRM routing engine settings.
type OSRMConfig struct {
	BaseURL string        `env:"OSRM_BASE_URL" env-default:"http://router.project-osrm.org"`
	Timeout time.Duration `env:"OSRM_TIMEOUT" env-default:"5s"`
}

// LogConfig specifies logger settings.
type LogConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
}

// LoadConfig reads configuration from environment variables and defaults.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read environment config: %w", err)
	}
	return &cfg, nil
}
