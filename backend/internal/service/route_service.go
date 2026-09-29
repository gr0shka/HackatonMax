package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"HackatonMax/internal/clients/ml"
	"HackatonMax/internal/entity"
)

type routeService struct {
	userRepo       UserRepository
	placesProvider PlacesProvider
	mlClient       MLClient
	routingClient  RoutingClient
}

// NewRouteService creates a new RouteService orchestrator.
func NewRouteService(
	userRepo UserRepository,
	placesProvider PlacesProvider,
	mlClient MLClient,
	routingClient RoutingClient,
) RouteService {
	return &routeService{
		userRepo:       userRepo,
		placesProvider: placesProvider,
		mlClient:       mlClient,
		routingClient:  routingClient,
	}
}

// BuildRoute orchestrates profile gathering, place discovery, ML ranking, and pedestrian path calculation.
func (s *routeService) BuildRoute(ctx context.Context, params BuildRouteParams) (*entity.Route, error) {
	// 1. Validate input parameters
	if !params.Start.IsValid() || !params.Finish.IsValid() {
		return nil, fmt.Errorf("%w: invalid start or finish coordinates", ErrInvalidRouteReq)
	}
	if params.BudgetMinutes <= 0 {
		return nil, fmt.Errorf("%w: budget minutes must be positive", ErrInvalidRouteReq)
	}
	if params.TransportMode == "" {
		params.TransportMode = "walking"
	}
	validTransport := map[string]bool{"walking": true, "metro": true, "bus": true, "car": true}
	if !validTransport[params.TransportMode] {
		return nil, fmt.Errorf("%w: unsupported transport mode", ErrInvalidRouteReq)
	}
	if params.ArrivalBufferMin < 0 {
		params.ArrivalBufferMin = 0
	}
	if params.ArrivalBufferMin >= params.BudgetMinutes {
		return nil, fmt.Errorf("%w: arrival buffer must be smaller than the time budget", ErrInvalidRouteReq)
	}
	if len(params.UserIDs) == 0 {
		return nil, fmt.Errorf("%w: at least one user id is required", ErrInvalidRouteReq)
	}

	// 2. Fetch user profiles
	users, err := s.userRepo.GetUsersByIDs(ctx, params.UserIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user profiles: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("%w: no users found for the provided IDs", ErrUserNotFound)
	}

	// 3. Discover candidate places nearby
	midLat := (params.Start.Lat + params.Finish.Lat) / 2.0
	midLon := (params.Start.Lon + params.Finish.Lon) / 2.0
	directDist := params.Start.DistanceMeters(params.Finish)

	// Search radius is based on distance with a minimum of 1500m
	searchRadius := math.Max(1500.0, directDist*0.75)

	candidates, err := s.placesProvider.FindPlaces(ctx, midLat, midLon, searchRadius, "", 50)
	if err != nil {
		return nil, fmt.Errorf("failed to discover candidate places: %w", err)
	}

	// Fallback to start point search if midpoint returned no candidates
	if len(candidates) == 0 {
		candidates, err = s.placesProvider.FindPlaces(ctx, params.Start.Lat, params.Start.Lon, searchRadius, "", 50)
		if err != nil {
			return nil, fmt.Errorf("failed to search places around start point: %w", err)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoPlacesFound
	}

	// Map candidates for quick ID lookup
	candidateMap := make(map[string]entity.Place, len(candidates))
	mlCandidates := make([]ml.CandidatePlaceDTO, len(candidates))
	for i, c := range candidates {
		candidateMap[c.ExternalID] = c
		candidateMap[c.ID.String()] = c

		mlCandidates[i] = ml.CandidatePlaceDTO{
			ID:             c.ExternalID,
			Name:           c.Name,
			Category:       c.Category,
			Lat:            c.Lat,
			Lon:            c.Lon,
			Rating:         c.Rating,
			AvgDurationMin: c.AvgDurationMin,
		}
	}

	// 4. Build and send request to ML optimization service
	mlProfiles := make([]ml.UserProfileDTO, len(users))
	for i, u := range users {
		mlProfiles[i] = ml.UserProfileDTO{
			UserID:    u.ID.String(),
			Interests: u.Interests,
		}
	}

	mlReq := &ml.OptimizeRequest{
		BudgetMinutes: params.BudgetMinutes - params.ArrivalBufferMin,
		BudgetRub: params.BudgetRub,
		TransportMode: params.TransportMode,
		ArrivalBufferMin: params.ArrivalBufferMin,
		Start: ml.PointDTO{
			Lat: params.Start.Lat,
			Lon: params.Start.Lon,
		},
		Finish: ml.PointDTO{
			Lat: params.Finish.Lat,
			Lon: params.Finish.Lon,
		},
		UserProfiles:    mlProfiles,
		CandidatePlaces: mlCandidates,
	}

	mlResp, err := s.mlClient.OptimizeRoute(ctx, mlReq)
	if err != nil {
		return nil, fmt.Errorf("ml route optimization failed: %w", err)
	}

	// 5. Build coordinate array for OSRM pedestrian routing
	selectedOrderedPlaces := make([]struct {
		place            entity.Place
		order            int
		allocatedTimeMin int
	}, 0, len(mlResp.SelectedPlaces))

	routeWaypoints := make([]entity.LatLon, 0, len(mlResp.SelectedPlaces)+2)
	routeWaypoints = append(routeWaypoints, params.Start)

	totalAllocatedPlaceMin := 0
	totalEstimatedCostRub := 0
	for _, item := range mlResp.SelectedPlaces {
		if place, ok := candidateMap[item.PlaceID]; ok {
			selectedOrderedPlaces = append(selectedOrderedPlaces, struct {
				place            entity.Place
				order            int
				allocatedTimeMin int
			}{
				place:            place,
				order:            item.Order,
				allocatedTimeMin: item.AllocatedTimeMin,
			})
			routeWaypoints = append(routeWaypoints, place.Point())
			totalAllocatedPlaceMin += item.AllocatedTimeMin
			totalEstimatedCostRub += item.EstimatedCostRub
		}
	}
	routeWaypoints = append(routeWaypoints, params.Finish)

	// 6. Request actual walking geometry and duration from OSRM
	routingRes, err := s.routingClient.BuildRoute(ctx, routeWaypoints, params.TransportMode)
	if err != nil {
		return nil, fmt.Errorf("pedestrian routing calculation failed: %w", err)
	}

	// 7. Aggregate final route entity
	routePoints := make([]entity.RoutePoint, 0, len(routeWaypoints))

	// Start point
	routePoints = append(routePoints, entity.RoutePoint{
		Order:       0,
		Type:        entity.RoutePointTypeStart,
		Location:    params.Start,
		DurationMin: 0,
	})

	// POI stops
	for i, sp := range selectedOrderedPlaces {
		var distFromPrev, durFromPrev float64
		if i < len(routingRes.Legs) {
			distFromPrev = routingRes.Legs[i].DistanceMeters
			durFromPrev = routingRes.Legs[i].DurationSec / 60.0
		}

		pCopy := sp.place
		routePoints = append(routePoints, entity.RoutePoint{
			Order:                  sp.order,
			Type:                   entity.RoutePointTypePlace,
			Location:               pCopy.Point(),
			Place:                  &pCopy,
			DurationMin:            sp.allocatedTimeMin,
			DistanceFromPrevMeters: distFromPrev,
			DurationFromPrevMin:    durFromPrev,
		})
	}

	// Finish point
	var lastLegDist, lastLegDur float64
	if len(routingRes.Legs) > 0 {
		lastLeg := routingRes.Legs[len(routingRes.Legs)-1]
		lastLegDist = lastLeg.DistanceMeters
		lastLegDur = lastLeg.DurationSec / 60.0
	}

	routePoints = append(routePoints, entity.RoutePoint{
		Order:                  len(routePoints),
		Type:                   entity.RoutePointTypeFinish,
		Location:               params.Finish,
		DurationMin:            0,
		DistanceFromPrevMeters: lastLegDist,
		DurationFromPrevMin:    lastLegDur,
	})

	totalWalkingMin := int(math.Round(routingRes.TotalDurationSec / 60.0))
	switch params.TransportMode {
	case "metro":
		totalEstimatedCostRub += 65
	case "bus":
		totalEstimatedCostRub += 50
	case "car":
		transportCost := int(math.Round(routingRes.TotalDistanceMeters / 1000.0 * 25.0))
		if transportCost < 150 {
			transportCost = 150
		}
		totalEstimatedCostRub += transportCost
	}
	finalRoute := &entity.Route{
		ID:                  uuid.New(),
		Geometry:            routingRes.Geometry,
		Points:              routePoints,
		MatchScore:          mlResp.MatchScore,
		MatchReasons:        mlResp.MatchReasons,
		TotalDurationMin:    totalWalkingMin + totalAllocatedPlaceMin,
		TravelDurationMin:   totalWalkingMin,
		VisitDurationMin:    totalAllocatedPlaceMin,
		ArrivalBufferMin:    params.ArrivalBufferMin,
		EstimatedCostRub:    totalEstimatedCostRub,
		TransportMode:       params.TransportMode,
		TotalDistanceMeters: routingRes.TotalDistanceMeters,
		CreatedAt:           time.Now(),
	}

	return finalRoute, nil
}
