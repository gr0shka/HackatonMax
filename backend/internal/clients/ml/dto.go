package ml

// PointDTO represents geographic coordinates for start and finish points.
type PointDTO struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// UserProfileDTO represents an individual user's interest vector.
type UserProfileDTO struct {
	UserID    string             `json:"user_id"`
	Interests map[string]float64 `json:"interests"`
}

// CandidatePlaceDTO represents a POI candidate retrieved from places provider.
type CandidatePlaceDTO struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	Rating         float64 `json:"rating"`
	AvgDurationMin int     `json:"avg_duration_min"`
	EstimatedCostRub int   `json:"estimated_cost_rub"`
}

// OptimizeRequest defines the payload sent to the ML ranking service.
type OptimizeRequest struct {
	BudgetMinutes   int                 `json:"budget_minutes"`
	BudgetRub       int                 `json:"budget_rub"`
	TransportMode   string              `json:"transport_mode"`
	ArrivalBufferMin int                `json:"arrival_buffer_min"`
	Start           PointDTO            `json:"start"`
	Finish          PointDTO            `json:"finish"`
	UserProfiles    []UserProfileDTO    `json:"user_profiles"`
	CandidatePlaces []CandidatePlaceDTO `json:"candidate_places"`
}

// SelectedPlaceItem represents a place chosen by ML with its order and duration.
type SelectedPlaceItem struct {
	PlaceID          string `json:"place_id"`
	Order            int    `json:"order"`
	AllocatedTimeMin int    `json:"allocated_time_min"`
	EstimatedCostRub int    `json:"estimated_cost_rub"`
}

// OptimizeResponse defines the response returned by the ML ranking service.
type OptimizeResponse struct {
	SelectedPlaces        []SelectedPlaceItem `json:"selected_places"`
	TotalEstimatedMinutes int                 `json:"total_estimated_minutes"`
	MatchScore            float64             `json:"match_score"`
	MatchReasons          []string            `json:"match_reasons"`
}
