package entity

import "math"

// LatLon represents a geographic point with Latitude and Longitude (WGS 84).
type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// IsValid validates whether latitude and longitude are within standard coordinate bounds.
func (l LatLon) IsValid() bool {
	return l.Lat >= -90.0 && l.Lat <= 90.0 && l.Lon >= -180.0 && l.Lon <= 180.0
}

// DistanceMeters calculates the great-circle distance between two coordinates in meters
// using the Haversine formula.
func (l LatLon) DistanceMeters(other LatLon) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (other.Lat - l.Lat) * math.Pi / 180.0
	dLon := (other.Lon - l.Lon) * math.Pi / 180.0

	lat1 := l.Lat * math.Pi / 180.0
	lat2 := other.Lat * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
