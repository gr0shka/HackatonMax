package places_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/clients/places"
	"HackatonMax/internal/entity"
)

type mockPlaceRepo struct {
	mock.Mock
}

func (m *mockPlaceRepo) FindNearbyPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]entity.Place, error) {
	args := m.Called(ctx, lat, lon, radiusMeters, limit)
	return args.Get(0).([]entity.Place), args.Error(1)
}

func (m *mockPlaceRepo) FindNearbyPlacesByCategory(ctx context.Context, lat, lon float64, radiusMeters float64, category string, limit int) ([]entity.Place, error) {
	args := m.Called(ctx, lat, lon, radiusMeters, category, limit)
	return args.Get(0).([]entity.Place), args.Error(1)
}

func (m *mockPlaceRepo) GetPlaceByID(ctx context.Context, id uuid.UUID) (*entity.Place, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.Place), args.Error(1)
}

func (m *mockPlaceRepo) GetPlaceByExternalID(ctx context.Context, externalID string) (*entity.Place, error) {
	args := m.Called(ctx, externalID)
	return args.Get(0).(*entity.Place), args.Error(1)
}

func (m *mockPlaceRepo) SavePlace(ctx context.Context, place *entity.Place) error {
	args := m.Called(ctx, place)
	return args.Error(0)
}

func (m *mockPlaceRepo) SavePlaces(ctx context.Context, p []entity.Place) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func TestTwoGisClient_SuccessAndCache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "/3.0/items", r.URL.Path)
		assert.Equal(t, "test-key", r.URL.Query().Get("key"))
		assert.Equal(t, "кафе", r.URL.Query().Get("q"))

		resp := map[string]interface{}{
			"meta": map[string]interface{}{"code": 200},
			"result": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":           "2gis-123",
						"name":         "Тестовое Кафе",
						"address_name": "ул. Ленина, 1",
						"point": map[string]interface{}{
							"lat": 55.75,
							"lon": 37.61,
						},
						"rubrics": []map[string]interface{}{
							{"name": "кафе"},
						},
						"reviews": map[string]interface{}{
							"rating": 4.8,
						},
					},
				},
				"total": 1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := places.NewTwoGisClient(places.TwoGisConfig{
		BaseURL:     server.URL,
		APIKey:      "test-key",
		CacheTTL:    1 * time.Minute,
		CacheSize:   100,
		HTTPTimeout: 2 * time.Second,
	})

	ctx := context.Background()

	// 1. Initial request (cache miss)
	res1, err := client.FindPlaces(ctx, 55.75, 37.61, 1000, "кафе", 10)
	require.NoError(t, err)
	require.Len(t, res1, 1)
	assert.Equal(t, "2gis-123", res1[0].ExternalID)
	assert.Equal(t, "Тестовое Кафе", res1[0].Name)
	assert.Equal(t, 4.8, res1[0].Rating)
	assert.Equal(t, 1, requestCount)

	// 2. Second request with same params (cache hit -> no HTTP call)
	res2, err := client.FindPlaces(ctx, 55.75, 37.61, 1000, "кафе", 10)
	require.NoError(t, err)
	assert.Len(t, res2, 1)
	assert.Equal(t, 1, requestCount, "Expected cache hit without extra HTTP request")
}

func TestTwoGisClient_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"meta":{"code":429,"error":{"type":"quota_exceeded","message":"Rate limit reached"}}}`))
	}))
	defer server.Close()

	client := places.NewTwoGisClient(places.TwoGisConfig{
		BaseURL:     server.URL,
		APIKey:      "test-key",
		CacheTTL:    1 * time.Minute,
		CacheSize:   100,
		HTTPTimeout: 2 * time.Second,
	})

	ctx := context.Background()
	_, err := client.FindPlaces(ctx, 55.75, 37.61, 1000, "парк", 10)
	assert.ErrorIs(t, err, places.ErrRateLimited)
}

func TestFallbackPlacesProvider_GracefulDegradation(t *testing.T) {
	// Setup 2GIS server returning 429 rate limit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	twoGis := places.NewTwoGisClient(places.TwoGisConfig{
		BaseURL:  server.URL,
		APIKey:   "test-key",
		CacheTTL: 1 * time.Minute,
	})

	mockRepo := new(mockPlaceRepo)
	fallbackPlaces := []entity.Place{
		{
			ID:         uuid.New(),
			ExternalID: "osm-999",
			Name:       "Локальный Парк (Fallback)",
			Category:   "парк",
			Lat:        55.75,
			Lon:        37.61,
		},
	}

	mockRepo.On("FindNearbyPlacesByCategory", mock.Anything, 55.75, 37.61, 1000.0, "парк", 10).
		Return(fallbackPlaces, nil)

	fallbackProvider := places.NewFallbackPlacesProvider(twoGis, mockRepo, nil)

	ctx := context.Background()
	res, err := fallbackProvider.FindPlaces(ctx, 55.75, 37.61, 1000, "парк", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "osm-999", res[0].ExternalID)
	assert.Equal(t, "Локальный Парк (Fallback)", res[0].Name)

	mockRepo.AssertExpectations(t)
}
