package main

import (
	"go.uber.org/fx"

	_ "HackatonMax/docs"
	"HackatonMax/internal/app"
)

// @title           Team Route Optimizer API
// @version         1.0
// @description     Backend-сервис оркестрации персонализированных пешеходных маршрутов
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

func main() {
	fx.New(app.Module).Run()
}
