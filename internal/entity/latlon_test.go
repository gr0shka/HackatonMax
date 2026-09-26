package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatLon_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		point LatLon
		valid bool
	}{
		{"valid moscow point", LatLon{Lat: 55.7558, Lon: 37.6173}, true},
		{"invalid latitude high", LatLon{Lat: 95.0, Lon: 37.0}, false},
		{"invalid latitude low", LatLon{Lat: -95.0, Lon: 37.0}, false},
		{"invalid longitude high", LatLon{Lat: 55.0, Lon: 185.0}, false},
		{"invalid longitude low", LatLon{Lat: 55.0, Lon: -185.0}, false},
		{"boundary zero", LatLon{Lat: 0.0, Lon: 0.0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.point.IsValid())
		})
	}
}

func TestLatLon_DistanceMeters(t *testing.T) {
	// Moscow Red Square (55.7539, 37.6208) to Bolshoi Theatre (55.7601, 37.6186)
	p1 := LatLon{Lat: 55.7539, Lon: 37.6208}
	p2 := LatLon{Lat: 55.7601, Lon: 37.6186}

	dist := p1.DistanceMeters(p2)
	// Approximate straight-line distance is ~700 meters
	assert.Greater(t, dist, 600.0)
	assert.Less(t, dist, 800.0)

	// Distance to itself must be 0
	assert.Equal(t, 0.0, p1.DistanceMeters(p1))
}
