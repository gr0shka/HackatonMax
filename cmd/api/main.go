package main

import (
	"go.uber.org/fx"

	_ "HackatonMax/docs"
	"HackatonMax/internal/app"
)

// @title           API Оптимизации Туристических Маршрутов
// @version         1.0.0
// @description     Сервис построения персонализированных групповых пешеходных маршрутов с учетом пересечения интересов участников, бюджета времени, интеграцией геоданных (2GIS / OpenStreetMap) и пешеходного графа OSRM.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Команда разработки маршрутов
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

func main() {
	fx.New(app.Module).Run()
}
