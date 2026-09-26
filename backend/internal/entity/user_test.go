package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_JSONSerialization(t *testing.T) {
	id := uuid.New()
	now := time.Now().Truncate(time.Second)

	user := User{
		ID:    id,
		Name:  "Алексей",
		Email: "alex@example.com",
		Interests: Interests{
			"cafe":    0.8,
			"museum":  0.6,
			"culture": 0.9,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var restored User
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, user.ID, restored.ID)
	assert.Equal(t, user.Name, restored.Name)
	assert.Equal(t, user.Email, restored.Email)
	assert.Equal(t, user.Interests["cafe"], restored.Interests["cafe"])
	assert.Equal(t, user.Interests["museum"], restored.Interests["museum"])
}

func TestInterests_DriverValuerAndScanner(t *testing.T) {
	interests := Interests{
		"cafe": 0.8,
		"park": 0.5,
	}

	val, err := interests.Value()
	require.NoError(t, err)

	var scanned Interests
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, 0.8, scanned["cafe"])
	assert.Equal(t, 0.5, scanned["park"])

	// Test nil and empty scan
	var empty Interests
	err = empty.Scan(nil)
	require.NoError(t, err)
	assert.NotNil(t, empty)

	err = empty.Scan("")
	require.NoError(t, err)
	assert.NotNil(t, empty)

	// Test invalid type
	err = empty.Scan(123)
	assert.Error(t, err)
}

func TestInterests_Merge(t *testing.T) {
	a := Interests{"cafe": 0.8, "park": 0.4}
	b := Interests{"cafe": 0.6, "museum": 0.9}

	merged := a.Merge(b)
	assert.Equal(t, 0.7, merged["cafe"]) // (0.8 + 0.6) / 2
	assert.Equal(t, 0.4, merged["park"])
	assert.Equal(t, 0.9, merged["museum"])
}

func TestInterests_CosineSimilarity(t *testing.T) {
	a := Interests{"cafe": 1.0, "park": 0.0}
	b := Interests{"cafe": 1.0, "park": 0.0}
	assert.InDelta(t, 1.0, a.CosineSimilarity(b), 0.0001)

	c := Interests{"cafe": 0.0, "park": 1.0}
	assert.InDelta(t, 0.0, a.CosineSimilarity(c), 0.0001)

	empty := Interests{}
	assert.Equal(t, 0.0, a.CosineSimilarity(empty))
}
