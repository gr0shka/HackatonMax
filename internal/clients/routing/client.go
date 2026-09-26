package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"HackatonMax/internal/entity"
)

// OSRMConfig holds options for the OSRM foot routing client.
type OSRMConfig struct {
	BaseURL     string
	HTTPTimeout time.Duration
}

// OSRMClient handles pedestrian route calculation using the OSRM HTTP API.
type OSRMClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOSRMClient creates a new OSRMClient.
func NewOSRMClient(cfg OSRMConfig) *OSRMClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://router.project-osrm.org"
	}

	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &OSRMClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type osrmRouteResponse struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Routes  []struct {
		Geometry struct {
			Type        string       `json:"type"`
			Coordinates [][2]float64 `json:"coordinates"`
		} `json:"geometry"`
		Duration float64 `json:"duration"`
		Distance float64 `json:"distance"`
		Legs     []struct {
			Duration float64 `json:"duration"`
			Distance float64 `json:"distance"`
		} `json:"legs"`
	} `json:"routes"`
}

// BuildFootRoute requests pedestrian route geometry and duration across ordered points.
func (c *OSRMClient) BuildFootRoute(ctx context.Context, points []entity.LatLon) (*RouteResult, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("at least 2 points are required to calculate a route, got %d", len(points))
	}

	coordParts := make([]string, len(points))
	for i, pt := range points {
		// OSRM requires {lon},{lat} format
		coordParts[i] = fmt.Sprintf("%.6f,%.6f", pt.Lon, pt.Lat)
	}

	coordsString := strings.Join(coordParts, ";")
	endpoint := fmt.Sprintf("%s/route/v1/foot/%s?overview=full&geometries=geojson", c.baseURL, coordsString)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSRM request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OSRM request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OSRM response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSRM returned status %d: %s", resp.StatusCode, string(body))
	}

	var data osrmRouteResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to decode OSRM json: %w", err)
	}

	if data.Code != "Ok" || len(data.Routes) == 0 {
		return nil, fmt.Errorf("OSRM routing error: code=%s, message=%s", data.Code, data.Message)
	}

	route := data.Routes[0]
	legs := make([]Leg, len(route.Legs))
	for i, l := range route.Legs {
		legs[i] = Leg{
			DurationSec:    l.Duration,
			DistanceMeters: l.Distance,
		}
	}

	geom := &entity.GeoJSONGeometry{
		Type:        route.Geometry.Type,
		Coordinates: route.Geometry.Coordinates,
	}

	return &RouteResult{
		Geometry:            geom,
		TotalDurationSec:    route.Duration,
		TotalDistanceMeters: route.Distance,
		Legs:                legs,
	}, nil
}
