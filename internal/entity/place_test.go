package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlace_JSONSerializationAndMethods(t *testing.T) {
	id := uuid.New()
	now := time.Now().Truncate(time.Second)

	place := Place{
		ID:             id,
		ExternalID:     "70000001029535674",
		Name:           "Кафе Пушкинъ",
		Address:        "Тверской бульвар, 26А",
		Lat:            55.7638,
		Lon:            37.6047,
		Category:       "кафе",
		Rating:         4.9,
		AvgDurationMin: 60,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	data, err := json.Marshal(place)
	require.NoError(t, err)

	var restored Place
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, place.ID, restored.ID)
	assert.Equal(t, place.ExternalID, restored.ExternalID)
	assert.Equal(t, place.Name, restored.Name)
	assert.Equal(t, place.Address, restored.Address)
	assert.Equal(t, place.Point(), LatLon{Lat: 55.7638, Lon: 37.6047})

	dist := place.DistanceTo(LatLon{Lat: 55.7638, Lon: 37.6047})
	assert.Equal(t, 0.0, dist)
}
