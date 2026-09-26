package dto

import "time"

// CreateOrUpdateUserRequest defines the request body for creating or updating a user.
type CreateOrUpdateUserRequest struct {
	Name      string             `json:"name" example:"Алексей" binding:"required"`
	Email     string             `json:"email" example:"alexey@example.com" binding:"required"`
	Interests map[string]float64 `json:"interests" example:"coffee:0.9,parks:0.7,art:0.4"`
}

// UserResponse defines the user profile response body.
type UserResponse struct {
	ID        string             `json:"id" example:"a4d3f56b-3cb8-45a7-96a9-83bc815b8b92"`
	Name      string             `json:"name" example:"Алексей"`
	Email     string             `json:"email" example:"alexey@example.com"`
	Interests map[string]float64 `json:"interests"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// AddFriendRequest defines the payload to establish a friendship link.
type AddFriendRequest struct {
	FriendID string `json:"friend_id" example:"b8c4d21e-12ab-4ef7-8910-123456789abc"`
}
