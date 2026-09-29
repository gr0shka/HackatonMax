package dto

import "HackatonMax/internal/entity"

// PointDTO представляет географические координаты точки на карте (широта и долгота WGS84).
type PointDTO struct {
	// Lat - географическая широта (latitude)
	Lat float64 `json:"lat" example:"55.148992"`
	// Lon - географическая долгота (longitude)
	Lon float64 `json:"lon" example:"61.376862"`
}

// BuildRouteRequest параметры для построения и оптимизации группового пешеходного маршрута.
type BuildRouteRequest struct {
	// BudgetMinutes - доступный лимит времени на маршрут в минутах
	BudgetMinutes int `json:"budget_minutes" example:"120" binding:"required"`
	// BudgetRub - максимальный бюджет на посещение мест, 0 означает без ограничения
	BudgetRub int `json:"budget_rub" example:"1000"`
	// TransportMode - walking, metro, bus или car
	TransportMode string `json:"transport_mode" example:"walking"`
	// ArrivalBufferMin - обязательный запас до дедлайна
	ArrivalBufferMin int `json:"arrival_buffer_min" example:"10"`
	// UserIDs - список UUID участников группы
	UserIDs []string `json:"user_ids" example:"a949a4f7-5a00-4178-b740-4ce831159fbb,484d5806-4d9a-444e-8a51-2f92034e8376" binding:"required"`
	// Start - географические координаты точки старта
	Start PointDTO `json:"start" binding:"required"`
	// Finish - географические координаты точки финиша
	Finish PointDTO `json:"finish" binding:"required"`
}

// RoutePointResponse представляет контрольную точку (start, place, finish) маршрута.
type RoutePointResponse struct {
	// Order - порядковый номер остановки на маршруте (начиная с 0)
	Order int `json:"order" example:"1"`
	// Type - тип точки: start (старт), place (локация), finish (финиш)
	Type string `json:"type" example:"place"`
	// Location - координаты точки
	Location PointDTO `json:"location"`
	// PlaceName - название заведения или локации (для промежуточных точек)
	PlaceName string `json:"place_name,omitempty" example:"Кофейня Зерно"`
	// Category - категория места (coffee, food, parks, art, sightseeing, bar)
	Category string `json:"category,omitempty" example:"coffee"`
	// DurationMin - время нахождения в локации (минуты)
	DurationMin int `json:"duration_min" example:"25"`
	// DistanceFromPrevMeters - пешеходное расстояние от предыдущей точки (метры)
	DistanceFromPrevMeters float64 `json:"distance_from_prev_meters,omitempty" example:"450.0"`
	// DurationFromPrevMin - время перехода от предыдущей точки (минуты)
	DurationFromPrevMin float64 `json:"duration_from_prev_min,omitempty" example:"6.5"`
}

// BuildRouteResponse представляет сгенерированный и оптимизированный групповой пешеходный маршрут.
type BuildRouteResponse struct {
	// RouteID - уникальный UUID сгенерированного маршрута
	RouteID string `json:"route_id" example:"e2b5e28a-6950-482a-aef2-ecbb903ca914"`
	// MatchScore - оценка удовлетворенности группы интересами выбранных мест (от 0.0 до 1.0)
	MatchScore float64 `json:"match_score" example:"0.89"`
	// MatchReasons - список текстовых объяснений, почему выбраны эти локации
	MatchReasons []string `json:"match_reasons" example:"Высокое совпадение по кофе (0.85),Оба участника любят прогулочные зоны"`
	// TotalDurationMin - общее время маршрута (время переходов + время нахождения в локациях) в минутах
	TotalDurationMin int `json:"total_duration_min" example:"85"`
	TravelDurationMin int `json:"travel_duration_min" example:"30"`
	VisitDurationMin int `json:"visit_duration_min" example:"45"`
	ArrivalBufferMin int `json:"arrival_buffer_min" example:"10"`
	EstimatedCostRub int `json:"estimated_cost_rub" example:"900"`
	TransportMode string `json:"transport_mode" example:"walking"`
	// TotalDistanceMeters - суммарная длина пешеходного трека в метрах
	TotalDistanceMeters float64 `json:"total_distance_meters" example:"2200.0"`
	// GeoJSON - стандартный GeoJSON Feature (LineString с массивом координат [lon, lat] для отрисовки линии на карте)
	GeoJSON entity.GeoJSONFeature `json:"geojson"`
	// Waypoints - упорядоченный список контрольных точек (start, place, finish) с временем на посещение и расстоянием от предыдущей точки
	Waypoints []RoutePointResponse `json:"waypoints"`
}
