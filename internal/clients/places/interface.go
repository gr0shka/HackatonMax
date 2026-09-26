package places

import (
	"context"
	"errors"

	"HackatonMax/internal/entity"
)

var (
	ErrRateLimited = errors.New("rate limit exceeded from places provider")
	ErrUnavailable = errors.New("places provider is temporarily unavailable")
)

// PlacesProvider defines the contract for searching and discovering nearby points of interest.
type PlacesProvider interface {
	FindPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, query string, limit int) ([]entity.Place, error)
}
