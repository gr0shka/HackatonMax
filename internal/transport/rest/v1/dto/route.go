package dto

import "HackatonMax/internal/entity"

// PointDTO represents latitude and longitude coordinates.
type PointDTO struct {
	Lat float64 `json:"lat" example:"55.751244"`
	Lon float64 `json:"lon" example:"37.618423"`
}

// BuildRouteRequest defines parameters required to generate an itinerary.
type BuildRouteRequest struct {
	Start         PointDTO `json:"start" binding:"required"`
	Finish        PointDTO `json:"finish" binding:"required"`
	BudgetMinutes int      `json:"budget_minutes" example:"120" binding:"required"`
	UserIDs       []string `json:"user_ids" example:"[\"a4d3f56b-3cb8-45a7-96a9-83bc815b8b92\"]" binding:"required"`
}

// RoutePointResponse represents an itinerary waypoint in API responses.
type RoutePointResponse struct {
	Order                  int      `json:"order"`
	Type                   string   `json:"type" example:"place"`
	Location               PointDTO `json:"location"`
	PlaceName              string   `json:"place_name,omitempty" example:"Кофейня Зерно"`
	Category               string   `json:"category,omitempty" example:"coffee"`
	DurationMin            int      `json:"duration_min" example:"25"`
	DistanceFromPrevMeters float64  `json:"distance_from_prev_meters,omitempty" example:"450.0"`
	DurationFromPrevMin    float64  `json:"duration_from_prev_min,omitempty" example:"6.5"`
}

// BuildRouteResponse represents the final route response returned to the frontend.
type BuildRouteResponse struct {
	RouteID             string                 `json:"route_id" example:"e2b5e28a-6950-482a-aef2-ecbb903ca914"`
	MatchScore          float64                `json:"match_score" example:"0.89"`
	MatchReasons        []string               `json:"match_reasons" example:"[\"Высокое совпадение по кофе (0.85)\"]"`
	TotalDurationMin    int                    `json:"total_duration_min" example:"85"`
	TotalDistanceMeters float64                `json:"total_distance_meters" example:"2200.0"`
	GeoJSON             entity.GeoJSONFeature  `json:"geojson"`
	Waypoints           []RoutePointResponse   `json:"waypoints"`
}
