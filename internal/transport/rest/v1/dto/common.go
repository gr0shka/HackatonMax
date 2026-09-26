package dto

// ErrorResponse стандартная структура ответа с ошибкой.
type ErrorResponse struct {
	// Error - краткое описание ошибки
	Error string `json:"error" example:"Невалидные входные данные"`
	// Details - детальное описание причины ошибки
	Details string `json:"details,omitempty" example:"Координаты старта находятся за пределами допустимого диапазона"`
}

// HealthResponse статус доступности и работоспособности сервиса.
type HealthResponse struct {
	// Status - статус доступности сервиса (ok)
	Status string `json:"status" example:"ok"`
}
