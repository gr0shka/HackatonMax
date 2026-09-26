package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
	"HackatonMax/internal/transport/rest/v1/dto"
)

// BuildRoute godoc
// @Summary      Build personalized group pedestrian route
// @Description  Generates an optimized walking route with ordered stops, GeoJSON geometry, duration, and match reasons
// @Tags         routes
// @Accept       json
// @Produce      json
// @Param        request body dto.BuildRouteRequest true "Route parameters"
// @Success      200  {object}  dto.BuildRouteResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/routes/build [post]
func (h *Handler) BuildRoute(w http.ResponseWriter, r *http.Request) {
	var req dto.BuildRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json payload", err.Error())
		return
	}

	if len(req.UserIDs) == 0 {
		respondError(w, http.StatusBadRequest, "user_ids must contain at least one user", "")
		return
	}

	userUUIDs := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, idStr := range req.UserIDs {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user id in list", idStr)
			return
		}
		userUUIDs = append(userUUIDs, parsed)
	}

	params := service.BuildRouteParams{
		Start: entity.LatLon{
			Lat: req.Start.Lat,
			Lon: req.Start.Lon,
		},
		Finish: entity.LatLon{
			Lat: req.Finish.Lat,
			Lon: req.Finish.Lon,
		},
		BudgetMinutes: req.BudgetMinutes,
		UserIDs:       userUUIDs,
	}

	route, err := h.routeService.BuildRoute(r.Context(), params)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRouteReq) {
			respondError(w, http.StatusBadRequest, "invalid route request", err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user profile not found", err.Error())
			return
		}
		if errors.Is(err, service.ErrNoPlacesFound) {
			respondError(w, http.StatusNotFound, "no places found for route", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to build route", err.Error())
		return
	}

	waypoints := make([]dto.RoutePointResponse, 0, len(route.Points))
	for _, pt := range route.Points {
		placeName := ""
		category := ""
		if pt.Place != nil {
			placeName = pt.Place.Name
			category = pt.Place.Category
		}

		waypoints = append(waypoints, dto.RoutePointResponse{
			Order: pt.Order,
			Type:  string(pt.Type),
			Location: dto.PointDTO{
				Lat: pt.Location.Lat,
				Lon: pt.Location.Lon,
			},
			PlaceName:              placeName,
			Category:               category,
			DurationMin:            pt.DurationMin,
			DistanceFromPrevMeters: pt.DistanceFromPrevMeters,
			DurationFromPrevMin:    pt.DurationFromPrevMin,
		})
	}

	resp := dto.BuildRouteResponse{
		RouteID:             route.ID.String(),
		MatchScore:          route.MatchScore,
		MatchReasons:        route.MatchReasons,
		TotalDurationMin:    route.TotalDurationMin,
		TotalDistanceMeters: route.TotalDistanceMeters,
		GeoJSON:             route.ToGeoJSON(),
		Waypoints:           waypoints,
	}

	respondJSON(w, http.StatusOK, resp)
}
