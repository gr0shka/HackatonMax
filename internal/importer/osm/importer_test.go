package osm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/importer/osm"
)

const sampleOverpassJSON = `{
  "version": 0.6,
  "generator": "Overpass API",
  "elements": [
    {
      "type": "node",
      "id": 101,
      "lat": 55.7535,
      "lon": 37.6212,
      "tags": {
        "name": "Кофейня Зерно",
        "amenity": "cafe",
        "addr:street": "ул. Никольская",
        "addr:housenumber": "5"
      }
    },
    {
      "type": "node",
      "id": 102,
      "lat": 55.7540,
      "lon": 37.6220,
      "tags": {
        "amenity": "cafe"
      }
    },
    {
      "type": "way",
      "id": 201,
      "center": {
        "lat": 55.7512,
        "lon": 37.6184
      },
      "tags": {
        "name": "Парк Зарядье",
        "leisure": "park"
      }
    },
    {
      "type": "way",
      "id": 202,
      "center": {
        "lat": 55.7551,
        "lon": 37.6243
      },
      "tags": {
        "name": "Третьяковская галерея",
        "tourism": "museum"
      }
    },
    {
      "type": "node",
      "id": 301,
      "lat": 55.7589,
      "lon": 37.6295,
      "tags": {
        "name": "Памятник Пушкину",
        "historic": "monument"
      }
    },
    {
      "type": "node",
      "id": 302,
      "lat": 0,
      "lon": 0,
      "tags": {
        "name": "Место без координат",
        "tourism": "attraction"
      }
    }
  ]
}`

func TestParseResponse_Success(t *testing.T) {
	places, totalFetched, filteredCount, err := osm.ParseResponse([]byte(sampleOverpassJSON), 10)
	require.NoError(t, err)

	assert.Equal(t, 6, totalFetched, "Total elements fetched from JSON")
	assert.Equal(t, 2, filteredCount, "Filtered out: 1 without name, 1 with zero coords")
	require.Len(t, places, 4, "Expected 4 valid places")

	// 1. Cafe
	p0 := places[0]
	assert.Equal(t, "osm_node_101", p0.ExternalID)
	assert.Equal(t, "Кофейня Зерно", p0.Name)
	assert.Equal(t, "ул. Никольская, 5", p0.Address)
	assert.Equal(t, "coffee", p0.Category)
	assert.Equal(t, 25, p0.AvgDurationMin)
	assert.GreaterOrEqual(t, p0.Rating, 4.2)
	assert.LessOrEqual(t, p0.Rating, 4.8)
	assert.InDelta(t, 55.7535, p0.Lat, 1e-4)
	assert.InDelta(t, 37.6212, p0.Lon, 1e-4)

	// 2. Park (Way with center coords)
	p1 := places[1]
	assert.Equal(t, "osm_way_201", p1.ExternalID)
	assert.Equal(t, "Парк Зарядье", p1.Name)
	assert.Equal(t, "parks", p1.Category)
	assert.Equal(t, 45, p1.AvgDurationMin)
	assert.InDelta(t, 55.7512, p1.Lat, 1e-4)
	assert.InDelta(t, 37.6184, p1.Lon, 1e-4)

	// 3. Museum
	p2 := places[2]
	assert.Equal(t, "osm_way_202", p2.ExternalID)
	assert.Equal(t, "Третьяковская галерея", p2.Name)
	assert.Equal(t, "art", p2.Category)
	assert.Equal(t, 60, p2.AvgDurationMin)

	// 4. Monument
	p3 := places[3]
	assert.Equal(t, "osm_node_301", p3.ExternalID)
	assert.Equal(t, "Памятник Пушкину", p3.Name)
	assert.Equal(t, "sightseeing", p3.Category)
	assert.Equal(t, 30, p3.AvgDurationMin)
}

func TestParseResponse_Limit(t *testing.T) {
	places, totalFetched, filteredCount, err := osm.ParseResponse([]byte(sampleOverpassJSON), 2)
	require.NoError(t, err)

	assert.Equal(t, 6, totalFetched)
	assert.Equal(t, 1, filteredCount) // Only node 102 was skipped before reaching limit of 2
	assert.Len(t, places, 2)
}

func TestClassifyCategory(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected string
	}{
		{"Cafe", map[string]string{"amenity": "cafe"}, "coffee"},
		{"Restaurant", map[string]string{"amenity": "restaurant"}, "food"},
		{"Fast food", map[string]string{"amenity": "fast_food"}, "food"},
		{"Bar", map[string]string{"amenity": "bar"}, "bar"},
		{"Pub", map[string]string{"amenity": "pub"}, "bar"},
		{"Park", map[string]string{"leisure": "park"}, "parks"},
		{"Garden", map[string]string{"leisure": "garden"}, "parks"},
		{"Museum", map[string]string{"tourism": "museum"}, "art"},
		{"Gallery", map[string]string{"tourism": "gallery"}, "art"},
		{"Attraction", map[string]string{"tourism": "attraction"}, "sightseeing"},
		{"Viewpoint", map[string]string{"tourism": "viewpoint"}, "sightseeing"},
		{"Monument", map[string]string{"historic": "monument"}, "sightseeing"},
		{"Memorial", map[string]string{"historic": "memorial"}, "sightseeing"},
		{"Coffee specialty", map[string]string{"amenity": "coffee_shop"}, "coffee"},
		{"Unknown", map[string]string{"random": "value"}, "sightseeing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, osm.ClassifyCategory(tt.tags))
		})
	}
}

func TestEstimateDuration(t *testing.T) {
	assert.Equal(t, 25, osm.EstimateDuration("coffee"))
	assert.Equal(t, 45, osm.EstimateDuration("parks"))
	assert.Equal(t, 60, osm.EstimateDuration("art"))
	assert.Equal(t, 45, osm.EstimateDuration("food"))
	assert.Equal(t, 40, osm.EstimateDuration("bar"))
	assert.Equal(t, 30, osm.EstimateDuration("sightseeing"))
	assert.Equal(t, 30, osm.EstimateDuration("other"))
}

func TestEstimateRating(t *testing.T) {
	for id := int64(0); id < 50; id++ {
		r := osm.EstimateRating(id)
		assert.GreaterOrEqual(t, r, 4.2)
		assert.LessOrEqual(t, r, 4.8)
	}
}

func TestResolveBBox(t *testing.T) {
	assert.Equal(t, osm.MoscowCenterBBox, osm.ResolveBBox("Moscow", ""))
	assert.Equal(t, osm.MoscowCenterBBox, osm.ResolveBBox("москва", ""))
	assert.Equal(t, "59.88,30.25,59.98,30.40", osm.ResolveBBox("SPb", ""))
	assert.Equal(t, "55.70,37.50,55.80,37.70", osm.ResolveBBox("", "55.70,37.50,55.80,37.70"))
	assert.Equal(t, "55.70,37.50,55.80,37.70", osm.ResolveBBox("55.70,37.50,55.80,37.70", ""))
}

func TestBuildQuery(t *testing.T) {
	bbox := "55.70,37.52,55.80,37.70"
	q := osm.BuildQuery(bbox, 500)
	assert.Contains(t, q, "[out:json][timeout:25];")
	assert.Contains(t, q, "amenity")
	assert.Contains(t, q, "tourism")
	assert.Contains(t, q, "leisure")
	assert.Contains(t, q, "historic")
	assert.Contains(t, q, "out center")
	assert.Contains(t, q, bbox)
}

func TestImporter_FetchOverpass_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("User-Agent"), "OSM-Importer")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleOverpassJSON))
	}))
	defer server.Close()

	imp := osm.NewImporter(osm.Config{
		OverpassURL: server.URL,
		HTTPTimeout: 2 * time.Second,
	}, nil, nil)

	data, err := imp.FetchOverpass(context.Background(), "test-query")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "Кофейня Зерно"))
}

func TestImporter_FetchOverpass_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte("504 Gateway Timeout"))
	}))
	defer server.Close()

	imp := osm.NewImporter(osm.Config{
		OverpassURL: server.URL,
		HTTPTimeout: 2 * time.Second,
	}, nil, nil)

	_, err := imp.FetchOverpass(context.Background(), "test-query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "504")
}
