package entity

import (
	"time"

	"github.com/google/uuid"
)

// RoutePointType indicates the functional role of a point in a route.
type RoutePointType string

const (
	RoutePointTypeStart  RoutePointType = "start"
	RoutePointTypePlace  RoutePointType = "place"
	RoutePointTypeFinish RoutePointType = "finish"
)

// RoutePoint represents an ordered waypoint in the planned itinerary.
type RoutePoint struct {
	Order                  int            `json:"order"`
	Type                   RoutePointType `json:"type"` // "start", "place", "finish"
	Location               LatLon         `json:"location"`
	Place                  *Place         `json:"place,omitempty"`
	DurationMin            int            `json:"duration_min"` // Time allocated to spend at this point
	DistanceFromPrevMeters float64        `json:"distance_from_prev_meters,omitempty"`
	DurationFromPrevMin    float64        `json:"duration_from_prev_min,omitempty"`
}

// GeoJSONGeometry represents GeoJSON LineString coordinates.
type GeoJSONGeometry struct {
	Type        string       `json:"type"`        // "LineString"
	Coordinates [][2]float64 `json:"coordinates"` // Array of [lon, lat] pairs
}

// GeoJSONFeature represents a standard GeoJSON Feature object.
type GeoJSONFeature struct {
	Type       string                 `json:"type"` // "Feature"
	Geometry   *GeoJSONGeometry       `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// Route represents the final orchestrated itinerary for a single user or group.
type Route struct {
	ID                  uuid.UUID        `json:"id"`
	Polyline            string           `json:"polyline,omitempty"`
	Geometry            *GeoJSONGeometry `json:"geometry,omitempty"`
	Points              []RoutePoint     `json:"points"`
	MatchScore          float64          `json:"match_score"`
	MatchReasons        []string         `json:"match_reasons,omitempty"`
	TotalDurationMin    int              `json:"total_duration_min"`
	TotalDistanceMeters float64          `json:"total_distance_meters"`
	CreatedAt           time.Time        `json:"created_at"`
}

// ToGeoJSON converts the Route into a standard GeoJSON Feature for frontend rendering.
func (r Route) ToGeoJSON() GeoJSONFeature {
	properties := map[string]interface{}{
		"id":                    r.ID.String(),
		"match_score":           r.MatchScore,
		"match_reasons":         r.MatchReasons,
		"total_duration_min":    r.TotalDurationMin,
		"total_distance_meters": r.TotalDistanceMeters,
		"points_count":          len(r.Points),
		"waypoints":             r.Points,
	}

	return GeoJSONFeature{
		Type:       "Feature",
		Geometry:   r.Geometry,
		Properties: properties,
	}
}
