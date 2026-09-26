package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoute_JSONAndGeoJSON(t *testing.T) {
	routeID := uuid.New()
	placeID := uuid.New()

	place := Place{
		ID:             placeID,
		ExternalID:     "70000001029535674",
		Name:           "Музей изобразительных искусств",
		Lat:            55.7473,
		Lon:            37.6051,
		Category:       "музей",
		Rating:         4.8,
		AvgDurationMin: 90,
	}

	points := []RoutePoint{
		{
			Order:       1,
			Type:        RoutePointTypeStart,
			Location:    LatLon{Lat: 55.7512, Lon: 37.6184},
			DurationMin: 0,
		},
		{
			Order:                  2,
			Type:                   RoutePointTypePlace,
			Location:               place.Point(),
			Place:                  &place,
			DurationMin:            90,
			DistanceFromPrevMeters: 1200,
			DurationFromPrevMin:    15,
		},
		{
			Order:                  3,
			Type:                   RoutePointTypeFinish,
			Location:               LatLon{Lat: 55.7441, Lon: 37.6055},
			DurationMin:            0,
			DistanceFromPrevMeters: 400,
			DurationFromPrevMin:    5,
		},
	}

	geometry := &GeoJSONGeometry{
		Type: "LineString",
		Coordinates: [][2]float64{
			{37.6184, 55.7512},
			{37.6051, 55.7473},
			{37.6055, 55.7441},
		},
	}

	route := Route{
		ID:                  routeID,
		Polyline:            "encoded_polyline_here",
		Geometry:            geometry,
		Points:              points,
		MatchScore:          0.92,
		TotalDurationMin:    110,
		TotalDistanceMeters: 1600,
		CreatedAt:           time.Now(),
	}

	data, err := json.Marshal(route)
	require.NoError(t, err)

	var restored Route
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, route.ID, restored.ID)
	assert.Equal(t, 3, len(restored.Points))
	assert.Equal(t, 0.92, restored.MatchScore)

	geoJSON := route.ToGeoJSON()
	assert.Equal(t, "Feature", geoJSON.Type)
	assert.Equal(t, geometry, geoJSON.Geometry)
	assert.Equal(t, 3, geoJSON.Properties["points_count"])
	assert.Equal(t, 0.92, geoJSON.Properties["match_score"])
}
