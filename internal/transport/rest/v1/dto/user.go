package dto

import "time"

// CreateOrUpdateUserRequest параметры тела запроса для создания или обновления профиля пользователя.
type CreateOrUpdateUserRequest struct {
	// Name - имя или никнейм пользователя
	Name string `json:"name" example:"Алексей" binding:"required"`
	// Email - уникальный адрес электронной почты
	Email string `json:"email" example:"alex@example.com" binding:"required"`
	// Interests - вектор интересов пользователя (coffee, art, parks, food, sightseeing, bar со значениями от 0.0 до 1.0)
	Interests map[string]float64 `json:"interests" example:"coffee:0.9,parks:0.8,art:0.2"`
}

// UserResponse структура данных профиля пользователя в ответах API.
type UserResponse struct {
	// ID - уникальный идентификатор пользователя (UUID)
	ID string `json:"id" example:"a949a4f7-5a00-4178-b740-4ce831159fbb"`
	// Name - имя пользователя
	Name string `json:"name" example:"Алексей"`
	// Email - адрес электронной почты
	Email string `json:"email" example:"alex@example.com"`
	// Interests - вектор интересов пользователя
	Interests map[string]float64 `json:"interests" example:"coffee:0.9,parks:0.8,art:0.2"`
	// CreatedAt - дата и время создания профиля
	CreatedAt time.Time `json:"created_at" example:"2026-09-26T18:00:00Z"`
	// UpdatedAt - дата и время последнего обновления профиля
	UpdatedAt time.Time `json:"updated_at" example:"2026-09-26T18:00:00Z"`
}

// AddFriendRequest параметры для добавления связи дружбы между пользователями.
type AddFriendRequest struct {
	// FriendID - UUID добавляемого друга
	FriendID string `json:"friend_id" example:"484d5806-4d9a-444e-8a51-2f92034e8376"`
}
