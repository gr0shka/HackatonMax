package places

import (
	"context"
	"fmt"
	"log/slog"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
)

// FallbackPlacesProvider wraps a primary provider (2GIS) with a local repository fallback (PostGIS/OSM).
type FallbackPlacesProvider struct {
	primary  PlacesProvider
	fallback service.PlaceRepository
	logger   *slog.Logger
}

// NewFallbackPlacesProvider creates a fault-tolerant PlacesProvider.
func NewFallbackPlacesProvider(primary PlacesProvider, fallback service.PlaceRepository, logger *slog.Logger) *FallbackPlacesProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &FallbackPlacesProvider{
		primary:  primary,
		fallback: fallback,
		logger:   logger,
	}
}

// FindPlaces attempts primary provider first, falling back to local PlaceRepository on error or rate-limit.
func (p *FallbackPlacesProvider) FindPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, query string, limit int) ([]entity.Place, error) {
	places, err := p.primary.FindPlaces(ctx, lat, lon, radiusMeters, query, limit)
	if err == nil && len(places) > 0 {
		// Populate local cache in background for future fallbacks
		go func(toSave []entity.Place) {
			bgCtx := context.Background()
			_ = p.fallback.SavePlaces(bgCtx, toSave)
		}(places)

		return places, nil
	}

	p.logger.WarnContext(ctx, "primary places provider failed, activating local PostGIS fallback",
		slog.Any("error", err),
		slog.Float64("lat", lat),
		slog.Float64("lon", lon),
		slog.Float64("radius", radiusMeters),
	)

	// Fallback to local DB repository
	var fallbackPlaces []entity.Place
	var fallbackErr error

	if query != "" {
		fallbackPlaces, fallbackErr = p.fallback.FindNearbyPlacesByCategory(ctx, lat, lon, radiusMeters, query, limit)
	}

	if len(fallbackPlaces) == 0 {
		fallbackPlaces, fallbackErr = p.fallback.FindNearbyPlaces(ctx, lat, lon, radiusMeters, limit)
	}

	if fallbackErr != nil {
		return nil, fmt.Errorf("both primary provider and local fallback failed: primary err: %v, fallback err: %w", err, fallbackErr)
	}

	return fallbackPlaces, nil
}
