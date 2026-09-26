package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"HackatonMax/internal/clients/ml"
	"HackatonMax/internal/clients/routing"
	"HackatonMax/internal/entity"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrPlaceNotFound   = errors.New("place not found")
	ErrInvalidRouteReq = errors.New("invalid route request parameters")
	ErrNoPlacesFound   = errors.New("no candidate places found in the search area")
)

// UserRepository defines persistence operations for user profiles and social links.
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error)
	CreateUser(ctx context.Context, user *entity.User) error
	UpdateUser(ctx context.Context, user *entity.User) error
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error)
}

// PlaceRepository defines persistence operations for places and spatial discovery.
type PlaceRepository interface {
	FindNearbyPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]entity.Place, error)
	FindNearbyPlacesByCategory(ctx context.Context, lat, lon float64, radiusMeters float64, category string, limit int) ([]entity.Place, error)
	GetPlaceByID(ctx context.Context, id uuid.UUID) (*entity.Place, error)
	GetPlaceByExternalID(ctx context.Context, externalID string) (*entity.Place, error)
	SavePlace(ctx context.Context, place *entity.Place) error
	SavePlaces(ctx context.Context, places []entity.Place) error
}

// PlacesProvider defines the contract for places discovery.
type PlacesProvider interface {
	FindPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, query string, limit int) ([]entity.Place, error)
}

// MLClient defines the contract for route optimization ranking.
type MLClient = ml.MLClient

// RoutingClient defines the contract for pedestrian route calculation.
type RoutingClient = routing.RoutingClient

// BuildRouteParams encapsulates all inputs for route orchestration.
type BuildRouteParams struct {
	Start         entity.LatLon `json:"start"`
	Finish        entity.LatLon `json:"finish"`
	BudgetMinutes int           `json:"budget_minutes"`
	UserIDs       []uuid.UUID   `json:"user_ids"`
}

// RouteService orchestrates route building between users, places, ML, and routing.
type RouteService interface {
	BuildRoute(ctx context.Context, params BuildRouteParams) (*entity.Route, error)
}

// UserService handles user lifecycle and interest profiles.
type UserService interface {
	GetUser(ctx context.Context, id uuid.UUID) (*entity.User, error)
	CreateOrUpdateUser(ctx context.Context, name, email string, interests entity.Interests) (*entity.User, error)
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error)
}
