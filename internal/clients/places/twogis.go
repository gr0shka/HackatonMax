package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"HackatonMax/internal/entity"
)

// TwoGisConfig contains configuration for 2GIS Places API adapter.
type TwoGisConfig struct {
	BaseURL     string
	APIKey      string
	RPS         float64
	Burst       int
	HTTPTimeout time.Duration
	CacheTTL    time.Duration
	CacheSize   int
}

// TwoGisClient connects to 2GIS Places API with client-side rate limiting and in-memory caching.
type TwoGisClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      *MemoryLRUTTLCache
}

// NewTwoGisClient creates an initialized TwoGisClient.
func NewTwoGisClient(cfg TwoGisConfig) *TwoGisClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://catalog.api.2gis.com"
	}

	rps := cfg.RPS
	if rps <= 0 {
		rps = 5.0
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = 5
	}

	httpTimeout := cfg.HTTPTimeout
	if httpTimeout <= 0 {
		httpTimeout = 5 * time.Second
	}

	return &TwoGisClient{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		cache:   NewMemoryLRUTTLCache(cfg.CacheSize, cfg.CacheTTL),
	}
}

type twoGisResponse struct {
	Meta struct {
		Code  int `json:"code"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"meta"`
	Result struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			AddressName string `json:"address_name"`
			Type        string `json:"type"`
			Point       *struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"point"`
			Rubrics []struct {
				Name string `json:"name"`
			} `json:"rubrics"`
			Reviews *struct {
				Rating float64 `json:"rating"`
			} `json:"reviews"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"result"`
}

// FindPlaces executes a search request against 2GIS Places API or returns cached results.
func (c *TwoGisClient) FindPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, query string, limit int) ([]entity.Place, error) {
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("%.4f:%.4f:%.0f:%s:%d", lat, lon, radiusMeters, query, limit)
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Respect client rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
	}

	endpoint := fmt.Sprintf("%s/3.0/items", c.baseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	// 2GIS catalog API format: location=lon,lat
	params.Set("location", fmt.Sprintf("%.6f,%.6f", lon, lat))
	params.Set("fields", "items.point,items.rubrics,items.reviews")

	// 2GIS Places API page_size must be between 1 and 10 (specifically enforced by demo key)
	pageSize := limit
	if pageSize <= 0 {
		pageSize = 10
	} else if pageSize > 10 {
		pageSize = 10
	}
	params.Set("page_size", strconv.Itoa(pageSize))

	// Search query q is required for /3.0/items circle search
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "достопримечательности"
	}
	params.Set("q", searchQuery)

	if radiusMeters > 0 {
		params.Set("radius", strconv.Itoa(int(radiusMeters)))
	}

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build 2gis request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read 2gis response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("2GIS API returned error", slog.Int("status", resp.StatusCode), slog.String("body", string(body)))
		return nil, fmt.Errorf("%w: 2gis returned status %d, body: %s", ErrUnavailable, resp.StatusCode, string(body))
	}

	var data twoGisResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse 2gis json response: %w", err)
	}

	if data.Meta.Code == 429 {
		return nil, ErrRateLimited
	}
	if data.Meta.Code != 200 && data.Meta.Code != 0 {
		slog.Error("2GIS API returned error", slog.Int("status", resp.StatusCode), slog.Int("meta_code", data.Meta.Code), slog.String("body", string(body)))
		errMsg := ""
		if data.Meta.Error != nil {
			errMsg = fmt.Sprintf(": %s (%s)", data.Meta.Error.Message, data.Meta.Error.Type)
		}
		return nil, fmt.Errorf("%w: 2gis meta code %d%s, body: %s", ErrUnavailable, data.Meta.Code, errMsg, string(body))
	}

	places := make([]entity.Place, 0, len(data.Result.Items))
	for _, item := range data.Result.Items {
		itemLat := lat
		itemLon := lon
		if item.Point != nil {
			itemLat = item.Point.Lat
			itemLon = item.Point.Lon
		}

		category := "attraction"
		if len(item.Rubrics) > 0 {
			category = item.Rubrics[0].Name
		}

		rating := 4.5
		if item.Reviews != nil && item.Reviews.Rating > 0 {
			rating = item.Reviews.Rating
		}

		place := entity.Place{
			ID:             uuid.New(),
			ExternalID:     item.ID,
			Name:           item.Name,
			Address:        item.AddressName,
			Category:       category,
			Rating:         rating,
			AvgDurationMin: 45,
			Lat:            itemLat,
			Lon:            itemLon,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		places = append(places, place)
	}

	c.cache.Set(cacheKey, places)
	return places, nil
}
