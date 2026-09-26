package routing

import (
	"context"

	"HackatonMax/internal/entity"
)

// Leg represents the transit between two consecutive waypoints in a calculated route.
type Leg struct {
	DurationSec    float64 `json:"duration_sec"`
	DistanceMeters float64 `json:"distance_meters"`
}

// RouteResult contains the aggregated geometry and metrics from the routing engine.
type RouteResult struct {
	Geometry            *entity.GeoJSONGeometry `json:"geometry"`
	TotalDurationSec    float64                 `json:"total_duration_sec"`
	TotalDistanceMeters float64                 `json:"total_distance_meters"`
	Legs                []Leg                   `json:"legs"`
}

// RoutingClient defines the contract for computing pedestrian routes and geometry.
type RoutingClient interface {
	BuildFootRoute(ctx context.Context, points []entity.LatLon) (*RouteResult, error)
}
