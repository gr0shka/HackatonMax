package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/clients/ml"
	"HackatonMax/internal/clients/routing"
	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
)

type mockPlacesProvider struct {
	mock.Mock
}

func (m *mockPlacesProvider) FindPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, query string, limit int) ([]entity.Place, error) {
	args := m.Called(ctx, lat, lon, radiusMeters, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Place), args.Error(1)
}

type mockMLClient struct {
	mock.Mock
}

func (m *mockMLClient) OptimizeRoute(ctx context.Context, req *ml.OptimizeRequest) (*ml.OptimizeResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ml.OptimizeResponse), args.Error(1)
}

type mockRoutingClient struct {
	mock.Mock
}

func (m *mockRoutingClient) BuildRoute(ctx context.Context, points []entity.LatLon, mode string) (*routing.RouteResult, error) {
	args := m.Called(ctx, points, mode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*routing.RouteResult), args.Error(1)
}

func TestRouteService_BuildRoute_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	placesProvider := new(mockPlacesProvider)
	mlClient := new(mockMLClient)
	routingClient := new(mockRoutingClient)

	routeSvc := service.NewRouteService(userRepo, placesProvider, mlClient, routingClient)

	u1 := uuid.New()
	u2 := uuid.New()
	users := []entity.User{
		{
			ID:        u1,
			Name:      "User 1",
			Interests: entity.Interests{"coffee": 0.9, "parks": 0.7},
		},
		{
			ID:        u2,
			Name:      "User 2",
			Interests: entity.Interests{"coffee": 0.4, "parks": 0.8},
		},
	}

	start := entity.LatLon{Lat: 55.751244, Lon: 37.618423}
	finish := entity.LatLon{Lat: 55.758900, Lon: 37.629500}

	p1 := entity.Place{
		ID:             uuid.New(),
		ExternalID:     "plc_501",
		Name:           "Кофейня Зерно",
		Category:       "coffee",
		Lat:            55.7535,
		Lon:            37.6212,
		Rating:         4.8,
		AvgDurationMin: 25,
	}

	p2 := entity.Place{
		ID:             uuid.New(),
		ExternalID:     "plc_742",
		Name:           "Арт-сквер",
		Category:       "art",
		Lat:            55.7551,
		Lon:            37.6243,
		Rating:         4.6,
		AvgDurationMin: 20,
	}

	// 1. Mock UserRepository
	userRepo.On("GetUsersByIDs", mock.Anything, []uuid.UUID{u1, u2}).Return(users, nil)

	// 2. Mock PlacesProvider
	placesProvider.On("FindPlaces", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), "", 50).
		Return([]entity.Place{p1, p2}, nil)

	// 3. Mock MLClient
	mlResp := &ml.OptimizeResponse{
		SelectedPlaces: []ml.SelectedPlaceItem{
			{PlaceID: "plc_501", Order: 1, AllocatedTimeMin: 25},
			{PlaceID: "plc_742", Order: 2, AllocatedTimeMin: 20},
		},
		TotalEstimatedMinutes: 85,
		MatchScore:            0.89,
		MatchReasons: []string{
			"Высокое совпадение по кофе (0.85)",
			"Оба участника любят прогулочные зоны",
		},
	}
	mlClient.On("OptimizeRoute", mock.Anything, mock.AnythingOfType("*ml.OptimizeRequest")).Return(mlResp, nil)

	// 4. Mock RoutingClient
	geom := &entity.GeoJSONGeometry{
		Type: "LineString",
		Coordinates: [][2]float64{
			{37.618423, 55.751244},
			{37.621200, 55.753500},
			{37.624300, 55.755100},
			{37.629500, 55.758900},
		},
	}
	routingRes := &routing.RouteResult{
		Geometry:            geom,
		TotalDurationSec:    1800.0, // 30 minutes walking
		TotalDistanceMeters: 2200.0,
		Legs: []routing.Leg{
			{DurationSec: 600, DistanceMeters: 700},
			{DurationSec: 500, DistanceMeters: 600},
			{DurationSec: 700, DistanceMeters: 900},
		},
	}
	routingClient.On("BuildRoute", mock.Anything, mock.AnythingOfType("[]entity.LatLon"), "walking").Return(routingRes, nil)

	params := service.BuildRouteParams{
		Start:         start,
		Finish:        finish,
		BudgetMinutes: 120,
		UserIDs:       []uuid.UUID{u1, u2},
	}

	route, err := routeSvc.BuildRoute(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, route)

	assert.Equal(t, 0.89, route.MatchScore)
	assert.Len(t, route.MatchReasons, 2)
	assert.Equal(t, 2200.0, route.TotalDistanceMeters)
	// 30 min walk + 25 min p1 + 20 min p2 = 75 min
	assert.Equal(t, 75, route.TotalDurationMin)
	assert.Len(t, route.Points, 4) // Start, p1, p2, Finish

	assert.Equal(t, entity.RoutePointTypeStart, route.Points[0].Type)
	assert.Equal(t, entity.RoutePointTypePlace, route.Points[1].Type)
	assert.Equal(t, "Кофейня Зерно", route.Points[1].Place.Name)
	assert.Equal(t, 25, route.Points[1].DurationMin)
	assert.Equal(t, entity.RoutePointTypePlace, route.Points[2].Type)
	assert.Equal(t, "Арт-сквер", route.Points[2].Place.Name)
	assert.Equal(t, 20, route.Points[2].DurationMin)
	assert.Equal(t, entity.RoutePointTypeFinish, route.Points[3].Type)

	// Test GeoJSON output
	geoJSON := route.ToGeoJSON()
	assert.Equal(t, "Feature", geoJSON.Type)
	assert.Equal(t, geom, geoJSON.Geometry)
	assert.Equal(t, 0.89, geoJSON.Properties["match_score"])
	assert.Equal(t, 4, geoJSON.Properties["points_count"])
}

func TestRouteService_BuildRoute_ValidationErrors(t *testing.T) {
	routeSvc := service.NewRouteService(nil, nil, nil, nil)

	// Invalid coordinates
	_, err := routeSvc.BuildRoute(context.Background(), service.BuildRouteParams{
		Start:         entity.LatLon{Lat: 100, Lon: 0},
		Finish:        entity.LatLon{Lat: 55, Lon: 37},
		BudgetMinutes: 60,
		UserIDs:       []uuid.UUID{uuid.New()},
	})
	assert.ErrorIs(t, err, service.ErrInvalidRouteReq)

	// Negative budget
	_, err = routeSvc.BuildRoute(context.Background(), service.BuildRouteParams{
		Start:         entity.LatLon{Lat: 55, Lon: 37},
		Finish:        entity.LatLon{Lat: 55, Lon: 37},
		BudgetMinutes: -10,
		UserIDs:       []uuid.UUID{uuid.New()},
	})
	assert.ErrorIs(t, err, service.ErrInvalidRouteReq)

	// Empty users
	_, err = routeSvc.BuildRoute(context.Background(), service.BuildRouteParams{
		Start:         entity.LatLon{Lat: 55, Lon: 37},
		Finish:        entity.LatLon{Lat: 55, Lon: 37},
		BudgetMinutes: 60,
		UserIDs:       []uuid.UUID{},
	})
	assert.ErrorIs(t, err, service.ErrInvalidRouteReq)
}

func TestRouteService_BuildRoute_NoPlacesFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	placesProvider := new(mockPlacesProvider)

	routeSvc := service.NewRouteService(userRepo, placesProvider, nil, nil)

	uID := uuid.New()
	userRepo.On("GetUsersByIDs", mock.Anything, []uuid.UUID{uID}).
		Return([]entity.User{{ID: uID}}, nil)

	placesProvider.On("FindPlaces", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]entity.Place{}, nil)

	params := service.BuildRouteParams{
		Start:         entity.LatLon{Lat: 55.75, Lon: 37.61},
		Finish:        entity.LatLon{Lat: 55.76, Lon: 37.62},
		BudgetMinutes: 60,
		UserIDs:       []uuid.UUID{uID},
	}

	_, err := routeSvc.BuildRoute(context.Background(), params)
	assert.ErrorIs(t, err, service.ErrNoPlacesFound)
}

func TestRouteService_BuildRoute_MLError(t *testing.T) {
	userRepo := new(mockUserRepo)
	placesProvider := new(mockPlacesProvider)
	mlClient := new(mockMLClient)

	routeSvc := service.NewRouteService(userRepo, placesProvider, mlClient, nil)

	uID := uuid.New()
	userRepo.On("GetUsersByIDs", mock.Anything, []uuid.UUID{uID}).
		Return([]entity.User{{ID: uID}}, nil)

	placesProvider.On("FindPlaces", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]entity.Place{{ID: uuid.New(), ExternalID: "p1"}}, nil)

	mlClient.On("OptimizeRoute", mock.Anything, mock.Anything).
		Return(nil, errors.New("ml unavailable"))

	params := service.BuildRouteParams{
		Start:         entity.LatLon{Lat: 55.75, Lon: 37.61},
		Finish:        entity.LatLon{Lat: 55.76, Lon: 37.62},
		BudgetMinutes: 60,
		UserIDs:       []uuid.UUID{uID},
	}

	_, err := routeSvc.BuildRoute(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ml route optimization failed")
}
