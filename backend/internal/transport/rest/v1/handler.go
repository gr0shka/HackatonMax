package v1

import (
	"encoding/json"
	"net/http"

	"HackatonMax/internal/service"
	"HackatonMax/internal/transport/rest/v1/dto"
)

// Handler coordinates all v1 HTTP endpoint controllers.
type Handler struct {
	routeService service.RouteService
	userService  service.UserService
}

// NewHandler creates a new Handler instance.
func NewHandler(routeService service.RouteService, userService service.UserService) *Handler {
	return &Handler{
		routeService: routeService,
		userService:  userService,
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func respondError(w http.ResponseWriter, status int, message string, details string) {
	respondJSON(w, status, dto.ErrorResponse{
		Error:   message,
		Details: details,
	})
}
