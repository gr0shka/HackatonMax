package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"HackatonMax/internal/entity"
)

// Default configurations and constants for OSM Seeder.
const (
	DefaultOverpassURL = "https://overpass-api.de/api/interpreter"
	DefaultCity        = "Moscow"
	DefaultLimit       = 500
	DefaultHTTPTimeout = 30 * time.Second
	DefaultUserAgent   = "OSM-Importer/1.0 (HackatonMax Route Optimizer)"
	DefaultBatchSize   = 100

	// MoscowCenterBBox covers central Moscow: Garden Ring, Boulevard Ring, Red Square,
	// Zaryadye, Chistye Prudy, Arbat, Patriarshiye Ponds, and surrounding cultural attractions.
	MoscowCenterBBox = "55.73,37.58,55.77,37.66"
)

// Predefined city bounding boxes (south,west,north,east).
var cityBBoxes = map[string]string{
	"moscow":           MoscowCenterBBox,
	"москва":           MoscowCenterBBox,
	"saint petersburg": "59.88,30.25,59.98,30.40",
	"санкт-петербург":  "59.88,30.25,59.98,30.40",
	"spb":              "59.88,30.25,59.98,30.40",
	"спб":              "59.88,30.25,59.98,30.40",
	"kazan":            "55.75,49.08,55.82,49.18",
	"казань":           "55.75,49.08,55.82,49.18",
	"novosibirsk":      "54.98,82.88,55.06,82.96",
	"новосибирск":      "54.98,82.88,55.06,82.96",
}

// Config configures the OSM Importer.
type Config struct {
	OverpassURL string
	City        string
	BBox        string
	Limit       int
	HTTPTimeout time.Duration
	UserAgent   string
	BatchSize   int
}

// Stats captures metrics from an import run.
type Stats struct {
	BBox          string        `json:"bbox"`
	City          string        `json:"city"`
	FetchedCount  int           `json:"fetched_count"`
	FilteredCount int           `json:"filtered_count"`
	ParsedCount   int           `json:"parsed_count"`
	InsertedCount int           `json:"inserted_count"`
	SkippedCount  int           `json:"skipped_count"`
	Duration      time.Duration `json:"duration"`
}

// Importer fetches POIs from OpenStreetMap via Overpass API and saves them into PostGIS.
type Importer struct {
	cfg        Config
	pool       *pgxpool.Pool
	httpClient *http.Client
	logger     *slog.Logger
}

// NewImporter creates a configured Importer.
func NewImporter(cfg Config, pool *pgxpool.Pool, logger *slog.Logger) *Importer {
	if cfg.OverpassURL == "" {
		cfg.OverpassURL = DefaultOverpassURL
	}
	if cfg.City == "" {
		cfg.City = DefaultCity
	}
	if cfg.Limit <= 0 {
		cfg.Limit = DefaultLimit
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = DefaultHTTPTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Importer{
		cfg:  cfg,
		pool: pool,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		logger: logger,
	}
}

// OverpassResponse represents the top-level JSON response from Overpass API.
type OverpassResponse struct {
	Version   float64           `json:"version"`
	Generator string            `json:"generator"`
	Elements  []OverpassElement `json:"elements"`
}

// OverpassElement represents a node or way returned by Overpass API.
type OverpassElement struct {
	Type   string            `json:"type"` // "node" or "way"
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat,omitempty"`
	Lon    float64           `json:"lon,omitempty"`
	Center *OverpassCenter   `json:"center,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// OverpassCenter holds computed centroid coordinates for a way or relation.
type OverpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Coordinates extracts latitude and longitude for both node and way elements.
func (e *OverpassElement) Coordinates() (float64, float64, bool) {
	if e.Type == "node" && e.Lat != 0 && e.Lon != 0 {
		return e.Lat, e.Lon, true
	}
	if e.Center != nil && e.Center.Lat != 0 && e.Center.Lon != 0 {
		return e.Center.Lat, e.Center.Lon, true
	}
	if e.Lat != 0 && e.Lon != 0 {
		return e.Lat, e.Lon, true
	}
	return 0, 0, false
}

// ResolveBBox resolves a bounding box string (south,west,north,east) from city name or bbox override.
func ResolveBBox(city, bbox string) string {
	if trimmed := strings.TrimSpace(bbox); trimmed != "" {
		return trimmed
	}
	trimmedCity := strings.TrimSpace(city)
	if strings.Contains(trimmedCity, ",") {
		return trimmedCity
	}
	if box, ok := cityBBoxes[strings.ToLower(trimmedCity)]; ok {
		return box
	}
	return MoscowCenterBBox
}

// BuildQuery constructs an Overpass QL query searching for leisure and tourism POIs.
func BuildQuery(bbox string, limit int) string {
	outLimit := ""
	if limit > 0 {
		// Ask Overpass for 2x limit to ensure enough named places remain after filtering
		outLimit = fmt.Sprintf(" %d", limit*2)
	}

	return fmt.Sprintf(`[out:json][timeout:25];
(
  node["amenity"~"^(cafe|restaurant|fast_food|bar)$"](%s);
  way["amenity"~"^(cafe|restaurant|fast_food|bar)$"](%s);
  node["tourism"~"^(museum|gallery|attraction|viewpoint)$"](%s);
  way["tourism"~"^(museum|gallery|attraction|viewpoint)$"](%s);
  node["leisure"~"^(park|garden)$"](%s);
  way["leisure"~"^(park|garden)$"](%s);
  node["historic"~"^(monument|memorial)$"](%s);
  way["historic"~"^(monument|memorial)$"](%s);
);
out center%s;`, bbox, bbox, bbox, bbox, bbox, bbox, bbox, bbox, outLimit)
}

// ClassifyCategory maps OSM tags to our application's high-level category string.
func ClassifyCategory(tags map[string]string) string {
	amenity := tags["amenity"]
	tourism := tags["tourism"]
	leisure := tags["leisure"]
	historic := tags["historic"]

	switch amenity {
	case "cafe":
		return "coffee"
	case "restaurant", "fast_food":
		return "food"
	case "bar", "pub":
		return "bar"
	}

	switch leisure {
	case "park", "garden":
		return "parks"
	}

	switch tourism {
	case "museum", "gallery":
		return "art"
	case "attraction", "viewpoint":
		return "sightseeing"
	}

	switch historic {
	case "monument", "memorial":
		return "sightseeing"
	}

	if strings.Contains(amenity, "coffee") {
		return "coffee"
	}

	return "sightseeing"
}

// EstimateDuration returns estimated visit duration in minutes based on place category.
func EstimateDuration(category string) int {
	switch category {
	case "coffee":
		return 25
	case "parks":
		return 45
	case "art":
		return 60
	case "food":
		return 45
	case "bar":
		return 40
	case "sightseeing":
		return 30
	default:
		return 30
	}
}

// EstimateRating generates a deterministic rating in the range [4.2, 4.8] from the OSM element ID.
func EstimateRating(id int64) float64 {
	absID := id
	if absID < 0 {
		absID = -absID
	}
	step := absID % 7 // 0..6
	return float64(42+step) / 10.0
}

// FormatAddress extracts street and house number from OSM tags if present.
func FormatAddress(tags map[string]string) string {
	street := tags["addr:street"]
	house := tags["addr:housenumber"]
	if street != "" && house != "" {
		return street + ", " + house
	}
	if street != "" {
		return street
	}
	return ""
}

// ParseResponse parses raw Overpass JSON bytes into entity.Place slice.
func ParseResponse(data []byte, limit int) ([]entity.Place, int, int, error) {
	var resp OverpassResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse overpass json: %w", err)
	}

	totalFetched := len(resp.Elements)
	filteredCount := 0
	places := make([]entity.Place, 0, len(resp.Elements))

	for _, elem := range resp.Elements {
		// 1. Skip objects without name tag
		name := strings.TrimSpace(elem.Tags["name"])
		if name == "" {
			filteredCount++
			continue
		}

		// 2. Validate coordinates
		lat, lon, ok := elem.Coordinates()
		if !ok || lat == 0 || lon == 0 {
			filteredCount++
			continue
		}

		category := ClassifyCategory(elem.Tags)
		duration := EstimateDuration(category)
		rating := EstimateRating(elem.ID)
		address := FormatAddress(elem.Tags)
		externalID := fmt.Sprintf("osm_%s_%d", elem.Type, elem.ID)

		place := entity.Place{
			ID:             uuid.New(),
			ExternalID:     externalID,
			Name:           name,
			Address:        address,
			Category:       category,
			Rating:         rating,
			AvgDurationMin: duration,
			Lat:            lat,
			Lon:            lon,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		places = append(places, place)
		if limit > 0 && len(places) >= limit {
			break
		}
	}

	return places, totalFetched, filteredCount, nil
}

// FetchOverpass executes the Overpass QL query and returns raw JSON bytes.
func (imp *Importer) FetchOverpass(ctx context.Context, query string) ([]byte, error) {
	maxAttempts := 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		form := url.Values{}
		form.Set("data", query)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, imp.cfg.OverpassURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create overpass request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", imp.cfg.UserAgent)

		imp.logger.Info("sending query to Overpass API",
			slog.String("url", imp.cfg.OverpassURL),
			slog.Duration("timeout", imp.cfg.HTTPTimeout),
			slog.Int("attempt", attempt),
		)

		start := time.Now()
		resp, err := imp.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("overpass request failed (possible timeout): %w", err)
			imp.logger.Warn("Overpass request failed", slog.Int("attempt", attempt), slog.Any("error", err))
			if attempt < maxAttempts {
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		imp.logger.Info("received response from Overpass API",
			slog.Int("status_code", resp.StatusCode),
			slog.Duration("latency", time.Since(start)),
		)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read overpass response body: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		lastErr = fmt.Errorf("overpass api returned status %d: %s", resp.StatusCode, string(body))
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < maxAttempts {
			imp.logger.Warn("Overpass returned server error, retrying...", slog.Int("status", resp.StatusCode), slog.Int("attempt", attempt))
			time.Sleep(2 * time.Second)
			continue
		}
		return nil, lastErr
	}

	return nil, lastErr
}

// SQL query for batch upsert with PostGIS geometry.
const insertPlaceQuery = `
INSERT INTO places (id, external_id, name, category, rating, avg_duration_min, geom)
VALUES ($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326))
ON CONFLICT (external_id) DO NOTHING;
`

// InsertPlacesBatch performs batched inserts into PostgreSQL using pgx.Batch.
func InsertPlacesBatch(ctx context.Context, pool *pgxpool.Pool, places []entity.Place, batchSize int, logger *slog.Logger) (int, int, error) {
	if len(places) == 0 {
		return 0, 0, nil
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	totalInserted := 0
	totalSkipped := 0

	for startIdx := 0; startIdx < len(places); startIdx += batchSize {
		endIdx := startIdx + batchSize
		if endIdx > len(places) {
			endIdx = len(places)
		}
		chunk := places[startIdx:endIdx]

		tx, err := pool.Begin(ctx)
		if err != nil {
			return totalInserted, totalSkipped, fmt.Errorf("failed to begin transaction: %w", err)
		}

		batch := &pgx.Batch{}
		for _, p := range chunk {
			batch.Queue(insertPlaceQuery,
				p.ID,
				p.ExternalID,
				p.Name,
				p.Category,
				p.Rating,
				p.AvgDurationMin,
				p.Lon, // $7 = Longitude
				p.Lat, // $8 = Latitude
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(chunk); i++ {
			ct, err := br.Exec()
			if err != nil {
				_ = br.Close()
				_ = tx.Rollback(ctx)
				return totalInserted, totalSkipped, fmt.Errorf("failed executing batch item %d: %w", i, err)
			}
			if ct.RowsAffected() > 0 {
				totalInserted++
			} else {
				totalSkipped++
			}
		}

		if err := br.Close(); err != nil {
			_ = tx.Rollback(ctx)
			return totalInserted, totalSkipped, fmt.Errorf("failed closing batch results: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return totalInserted, totalSkipped, fmt.Errorf("failed committing transaction: %w", err)
		}

		if logger != nil {
			logger.Info("batch progress",
				slog.Int("processed", endIdx),
				slog.Int("total", len(places)),
				slog.Int("inserted", totalInserted),
				slog.Int("skipped_duplicates", totalSkipped),
			)
		}
	}

	return totalInserted, totalSkipped, nil
}

// Run executes the complete OSM import flow: query generation, HTTP fetch, parsing, and DB upsert.
func (imp *Importer) Run(ctx context.Context) (*Stats, error) {
	startTime := time.Now()
	bbox := ResolveBBox(imp.cfg.City, imp.cfg.BBox)

	imp.logger.Info("starting OSM POI import",
		slog.String("city", imp.cfg.City),
		slog.String("bbox", bbox),
		slog.Int("limit", imp.cfg.Limit),
	)

	query := BuildQuery(bbox, imp.cfg.Limit)
	rawJSON, err := imp.FetchOverpass(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed fetching OSM data: %w", err)
	}

	places, fetchedCount, filteredCount, err := ParseResponse(rawJSON, imp.cfg.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed parsing OSM data: %w", err)
	}

	imp.logger.Info("OSM data parsed and filtered",
		slog.Int("fetched_from_osm", fetchedCount),
		slog.Int("filtered_out", filteredCount),
		slog.Int("valid_places_ready", len(places)),
	)

	var insertedCount, skippedCount int
	if imp.pool != nil && len(places) > 0 {
		insertedCount, skippedCount, err = InsertPlacesBatch(ctx, imp.pool, places, imp.cfg.BatchSize, imp.logger)
		if err != nil {
			return nil, fmt.Errorf("failed inserting places into database: %w", err)
		}
	}

	stats := &Stats{
		City:          imp.cfg.City,
		BBox:          bbox,
		FetchedCount:  fetchedCount,
		FilteredCount: filteredCount,
		ParsedCount:   len(places),
		InsertedCount: insertedCount,
		SkippedCount:  skippedCount,
		Duration:      time.Since(startTime),
	}

	imp.logger.Info("OSM POI import finished successfully",
		slog.Int("fetched", stats.FetchedCount),
		slog.Int("filtered", stats.FilteredCount),
		slog.Int("valid", stats.ParsedCount),
		slog.Int("inserted", stats.InsertedCount),
		slog.Int("skipped_duplicates", stats.SkippedCount),
		slog.Duration("elapsed", stats.Duration),
	)

	return stats, nil
}
