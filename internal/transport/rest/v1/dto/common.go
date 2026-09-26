package dto

// ErrorResponse represents an API error response body.
type ErrorResponse struct {
	Error   string `json:"error" example:"invalid request payload"`
	Details string `json:"details,omitempty" example:"start coordinates are outside valid bounds"`
}
