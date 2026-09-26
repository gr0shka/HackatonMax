package entity

import (
	"time"

	"github.com/google/uuid"
)

// Place represents a Point of Interest (POI) obtained from 2GIS or the fallback local database.
type Place struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ExternalID     string    `json:"external_id" db:"external_id"`
	Name           string    `json:"name" db:"name"`
	Address        string    `json:"address,omitempty" db:"address"`
	Lat            float64   `json:"lat" db:"lat"`
	Lon            float64   `json:"lon" db:"lon"`
	Category       string    `json:"category" db:"category"`
	Rating         float64   `json:"rating" db:"rating"`
	AvgDurationMin int       `json:"avg_duration_min" db:"avg_duration_min"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Point returns the LatLon coordinate of the Place.
func (p Place) Point() LatLon {
	return LatLon{
		Lat: p.Lat,
		Lon: p.Lon,
	}
}

// DistanceTo calculates distance in meters from this place to a coordinate.
func (p Place) DistanceTo(target LatLon) float64 {
	return p.Point().DistanceMeters(target)
}
