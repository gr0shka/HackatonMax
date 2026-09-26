package ml_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/clients/ml"
)

func TestClient_OptimizeRoute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/optimize", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req ml.OptimizeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, 120, req.BudgetMinutes)
		assert.Equal(t, 55.751244, req.Start.Lat)
		assert.Equal(t, 37.618423, req.Start.Lon)
		assert.Len(t, req.UserProfiles, 2)
		assert.Len(t, req.CandidatePlaces, 2)

		resp := ml.OptimizeResponse{
			SelectedPlaces: []ml.SelectedPlaceItem{
				{
					PlaceID:          "plc_501",
					Order:            1,
					AllocatedTimeMin: 25,
				},
				{
					PlaceID:          "plc_742",
					Order:            2,
					AllocatedTimeMin: 20,
				},
			},
			TotalEstimatedMinutes: 85,
			MatchScore:            0.89,
			MatchReasons: []string{
				"Высокое совпадение по кофе (0.85)",
				"Оба участника любят прогулочные зоны",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ml.NewClient(ml.Config{
		BaseURL: server.URL,
		Timeout: 2 * time.Second,
	})

	req := &ml.OptimizeRequest{
		BudgetMinutes: 120,
		Start: ml.PointDTO{
			Lat: 55.751244,
			Lon: 37.618423,
		},
		Finish: ml.PointDTO{
			Lat: 55.758900,
			Lon: 37.629500,
		},
		UserProfiles: []ml.UserProfileDTO{
			{
				UserID: "usr_101",
				Interests: map[string]float64{
					"coffee": 0.9,
					"parks":  0.7,
					"art":    0.2,
				},
			},
			{
				UserID: "usr_202",
				Interests: map[string]float64{
					"coffee": 0.4,
					"parks":  0.8,
					"art":    0.9,
				},
			},
		},
		CandidatePlaces: []ml.CandidatePlaceDTO{
			{
				ID:             "plc_501",
				Name:           "Кофейня Зерно",
				Category:       "coffee",
				Lat:            55.753500,
				Lon:            37.621200,
				Rating:         4.8,
				AvgDurationMin: 25,
			},
			{
				ID:             "plc_742",
				Name:           "Арт-сквер",
				Category:       "art",
				Lat:            55.755100,
				Lon:            37.624300,
				Rating:         4.6,
				AvgDurationMin: 20,
			},
		},
	}

	result, err := client.OptimizeRoute(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, result.SelectedPlaces, 2)
	assert.Equal(t, "plc_501", result.SelectedPlaces[0].PlaceID)
	assert.Equal(t, 25, result.SelectedPlaces[0].AllocatedTimeMin)
	assert.Equal(t, 85, result.TotalEstimatedMinutes)
	assert.Equal(t, 0.89, result.MatchScore)
	assert.Len(t, result.MatchReasons, 2)
}

func TestClient_OptimizeRoute_NilRequest(t *testing.T) {
	client := ml.NewClient(ml.Config{})
	_, err := client.OptimizeRoute(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestClient_OptimizeRoute_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"model inference failed"}`))
	}))
	defer server.Close()

	client := ml.NewClient(ml.Config{
		BaseURL: server.URL,
	})

	_, err := client.OptimizeRoute(context.Background(), &ml.OptimizeRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestClient_OptimizeRoute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := ml.NewClient(ml.Config{
		BaseURL: server.URL,
		Timeout: 20 * time.Millisecond,
	})

	_, err := client.OptimizeRoute(context.Background(), &ml.OptimizeRequest{})
	assert.Error(t, err)
}
