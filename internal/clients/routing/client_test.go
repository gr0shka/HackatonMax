package routing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/clients/routing"
	"HackatonMax/internal/entity"
)

func TestOSRMClient_BuildFootRoute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/route/v1/foot/")
		assert.Equal(t, "full", r.URL.Query().Get("overview"))
		assert.Equal(t, "geojson", r.URL.Query().Get("geometries"))

		resp := map[string]interface{}{
			"code": "Ok",
			"routes": []map[string]interface{}{
				{
					"duration": 600.5,
					"distance": 850.0,
					"geometry": map[string]interface{}{
						"type": "LineString",
						"coordinates": [][]float64{
							{37.61, 55.75},
							{37.62, 55.76},
						},
					},
					"legs": []map[string]interface{}{
						{
							"duration": 600.5,
							"distance": 850.0,
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := routing.NewOSRMClient(routing.OSRMConfig{
		BaseURL:     server.URL,
		HTTPTimeout: 2 * time.Second,
	})

	points := []entity.LatLon{
		{Lat: 55.75, Lon: 37.61},
		{Lat: 55.76, Lon: 37.62},
	}

	res, err := client.BuildFootRoute(context.Background(), points)
	require.NoError(t, err)
	assert.Equal(t, 600.5, res.TotalDurationSec)
	assert.Equal(t, 850.0, res.TotalDistanceMeters)
	assert.Equal(t, "LineString", res.Geometry.Type)
	assert.Len(t, res.Geometry.Coordinates, 2)
	assert.Len(t, res.Legs, 1)
}

func TestOSRMClient_BuildFootRoute_Validation(t *testing.T) {
	client := routing.NewOSRMClient(routing.OSRMConfig{})
	_, err := client.BuildFootRoute(context.Background(), []entity.LatLon{{Lat: 55.75, Lon: 37.61}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 points")
}

func TestOSRMClient_BuildFootRoute_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"Error","message":"internal server error"}`))
	}))
	defer server.Close()

	client := routing.NewOSRMClient(routing.OSRMConfig{
		BaseURL: server.URL,
	})

	points := []entity.LatLon{
		{Lat: 55.75, Lon: 37.61},
		{Lat: 55.76, Lon: 37.62},
	}

	_, err := client.BuildFootRoute(context.Background(), points)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OSRM returned status 500")
}
