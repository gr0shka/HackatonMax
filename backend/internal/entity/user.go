package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Interests represents a vector of user interests and their preference weights (category -> weight in [0.0, 1.0]).
type Interests map[string]float64

// Value implements driver.Valuer for PostgreSQL JSONB serialization.
func (i Interests) Value() (driver.Value, error) {
	if i == nil {
		return "{}", nil
	}
	return json.Marshal(i)
}

// Scan implements sql.Scanner for PostgreSQL JSONB deserialization.
func (i *Interests) Scan(src any) error {
	if src == nil {
		*i = make(Interests)
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for Interests scan: %T", src)
	}

	if len(data) == 0 {
		*i = make(Interests)
		return nil
	}

	temp := make(map[string]float64)
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal Interests jsonb: %w", err)
	}

	*i = temp
	return nil
}

// Merge combines multiple interest profiles by averaging the weights of common categories
// and preserving unique interests with fractional weight.
func (i Interests) Merge(other Interests) Interests {
	result := make(Interests)
	for k, v := range i {
		result[k] = v
	}

	for k, v := range other {
		if cur, ok := result[k]; ok {
			result[k] = (cur + v) / 2.0
		} else {
			result[k] = v
		}
	}
	return result
}

// CosineSimilarity computes cosine similarity between two interest vectors.
func (i Interests) CosineSimilarity(other Interests) float64 {
	if len(i) == 0 || len(other) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for k, vA := range i {
		normA += vA * vA
		if vB, ok := other[k]; ok {
			dotProduct += vA * vB
		}
	}
	for _, vB := range other {
		normB += vB * vB
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// User represents a system user profile.
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Interests Interests `json:"interests" db:"interests"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Friendship represents an established friendship link between two users.
type Friendship struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	FriendID  uuid.UUID `json:"friend_id" db:"friend_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
