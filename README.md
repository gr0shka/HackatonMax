Markdown

# 🧭 Team Route Optimizer Backend (Hackathon MVP)

## 📌 Project Overview
Backend-сервис оркестрации персонализированных пешеходных маршрутов для одного или группы пользователей на базе заданного тайм-лимита. 

Сервис агрегирует профили пользователей, запрашивает кандидатов POI через 2GIS Places API (с поддержкой In-Memory кэширования и фоллбека на локальную БД/OSM при rate-limit), передает контекст во внешний ML-сервис для ранжирования и сборки цепочки, запрашивает геометрию маршрута в OSRM и возвращает готовый GeoJSON на фронтенд.

---

## 🛑 CRITICAL AGENT RULES: ZERO HALLUCINATION & API BUDGET

1. **NO GUESSING / NO HALLUCINATIONS**:
   - Категорически запрещено выдумывать сигнатуры запросов, названия параметров, заголовки авторизации или формат ответа для 2GIS API, ML-сервиса или OSRM.
   - Если предоставленной в `docs/context/` документации недостаточно, параметр неочевиден или возникли сомнения в схеме данных — **НЕМЕДЛЕННО ОСТАНОВИСЬ И СПРОСИ У МЕНЯ**. 
   - Не пытайся «угадать» формат по аналогии с другими API. Сформулируй точный вопрос: какой эндпоинт, какой параметр или какой payload требуется уточнить.
2. **API BUDGET PROTECTION**:
   - Тестовый демо-ключ 2GIS имеет жесткие лимиты (RPS и суточную квоту).
   - **Запрещено** запускать код или тесты, делающие реальные сетевые запросы к 2GIS API методом проб и ошибок.
   - Любое тестирование интеграции с 2GIS на этапе разработки должно проводиться **исключительно через моки (`httptest.Server`)** и заготовленные JSON-фикстуры ответов.
   - Реальный запрос к внешнему API допускается строго после твоего подтверждения и успешного прохождения всех mock-тестов.

---

## 🏗 Architecture & Stack
- **Language**: Go 1.23+
- **Architecture**: Classic Layered Architecture (Transport -> Service/Usecase -> Repository/Clients)
- **DI Engine**: `go.uber.org/fx` (Uber Fx)
- **HTTP Router**: `github.com/go-chi/chi/v5`
- **Database**: PostgreSQL 16 + PostGIS
- **Migrations**: `pressly/goose/v3` (SQL migrations)
- **API Spec**: OpenAPI 3.0 via `swaggo/swag` (Swagger UI on `/swagger/*`)
- **Testing**:
  - Unit tests: `stretchr/testify` (mocking interfaces via `vektra/mockery` or testify/mock)
  - Integration DB tests: `testcontainers/testcontainers-go` (Postgres/PostGIS container)

---

## 📂 Required Project Structure
Строго придерживаться следующей структуры директорий:

```text
.
├── cmd/
│   └── api/
│       └── main.go              # Точка входа, fx.New(...).Run()
├── config/
│   └── config.go             # Конфигурация (cleanenv/viper) через ENV
├── docs/                     # Автогенерируемый Swagger (swag init)
│   ├── context/              # Документация внешних API и лимиты
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── app/
│   │   └── app.go            # fx.Module сборщик всех зависимостей
│   ├── transport/
│   │   └── rest/
│   │       ├── router.go     # Регистрация Chi маршрутов, CORS, Middleware
│   │       ├── v1/
│   │       │   ├── dto/      # Входные и выходные структуры REST API
│   │       │   ├── handler.go
│   │       │   ├── route.go  # Эндпоинты фронтенда: генерация маршрутов
│   │       │   └── user.go   # Эндпоинты управления профилями/друзьями
│   ├── clients/
│   │   ├── ml/               # HTTP-клиент к внешнему ML-сервису
│   │   │   ├── client.go
│   │   │   ├── dto.go        # Спецификация запроса/ответа ML
│   │   │   └── interface.go
│   │   ├── routing/          # HTTP-клиент к OSRM (пешеходные треки)
│   │   │   ├── client.go
│   │   │   └── interface.go
│   │   └── places/           # Провайдеры мест (2GIS + Fallback)
│   │       ├── interface.go  # PlacesProvider interface
│   │       ├── twogis.go     # Клиент 2GIS API (учитывать демо-лимиты и кэш)
│   │       └── fallback.go   # Резервный источник мест из БД/OSM
│   ├── service/              # Слой бизнес-логики (Usecases)
│   │   ├── interfaces.go     # Интерфейсы репозиториев и клиентов для изоляции
│   │   ├── route_service.go  # Оркестрация флоу маршрутов
│   │   └── user_service.go   # Управление пользователями и профилями
│   ├── repository/           # Слой персистентности (Postgres)
│   │   ├── postgres/
│   │   │   ├── user_repo.go
│   │   │   ├── place_repo.go
│   │   │   └── db.go         # pgxpool / database/sql подключение
│   └── entity/               # Чистые доменные модели (User, Place, Route, LatLon)
├── migrations/               # Goose SQL-миграции
│   └── 20260926000001_init.sql
├── docker-compose.yml        # Postgres + PostGIS
├── Makefile
└── README.md

🚦 Step-by-Step Implementation Instructions (Agent Protocol)

Выполняй разработку строго последовательно, этап за этапом. После каждого этапа код должен компилироваться (go build ./...).
Stage 1: Domain Entities & Migrations

    Опиши доменные структуры в internal/entity/:

        User: ID, Name, Email, Interests (Map/Vector весов string -> float64).

        Place: ID, ExternalID, Name, Lat, Lon, Category, Rating, AvgDurationMin.

        Route: Итоговый маршрут, полилиния, список точек, MatchScore.

    Создай начальную миграцию для Goose в migrations/:

        Включение расширения PostGIS: CREATE EXTENSION IF NOT EXISTS postgis;

        Таблица users (с JSONB полем interests).

        Таблица places с пространственной колонкой geom GEOMETRY(Point, 4326) и GiST-индексом.

        Таблица связей/друзей пользователей.

Stage 2: Repository Layer & Testcontainers

    Реализуй методы в internal/repository/postgres/:

        GetUserByID, GetUsersByIDs (для друзей).

        FindNearbyPlaces(lat, lon, radiusMeters, limit) с использованием ST_DWithin в PostGIS.

    Обязательно: Напиши интеграционные тесты для репозиториев с использованием testcontainers/testcontainers-go (образ postgis/postgis:16-3.4). Тесты должны поднимать контейнер, накатывать миграции через Goose и проверять выборку.

Stage 3: External Clients (ML, Routing, 2GIS)

    Places Provider (internal/clients/places/):

        Изучи файлы в docs/context/. Если параметров или структуры ответа недостаточно — запроси у меня.

        Реализуй интерфейс PlacesProvider.

        Напиши адаптер под 2GIS Places API с учетом лимитов (RPS, таймауты).

        Реализуй In-Memory LRU/TTL кэш перед вызовом API для предотвращения исчерпания демо-квоты.

        Реализуй отказоустойчивую обертку: если 2GIS отдает 429 или падает по таймауту, запрос идет в локальный PlaceRepository (OSM fallback).

    ML Client (internal/clients/ml/):

        HTTP-клиент к сервису ранжирования (POST /api/v1/optimize).

        Полноценная обработка таймаутов (http.Client с таймаутом не более 5с).

    OSRM Client (internal/clients/routing/):

        Вызов /route/v1/foot/{coords}?overview=full&geometries=geojson для получения полилинии и точного pedestrian duration.

    Покрой все клиенты модульными тестами с моками HTTP (httptest.Server). Не делай боевых запросов!

Stage 4: Service / Usecase Layer

    Реализуй RouteService:

        Принимает: точку старта, точку финиша, лимит времени в минутах, ID пользователей.

        Запрашивает профили участников через UserRepository.

        Запрашивает кандидатов заведений через PlacesProvider (2GIS / Fallback).

        Формирует payload и отправляет в MLClient.

        На базе отобранных точек запрашивает геометрию у RoutingClient.

        Агрегирует итоговый результат.

    Покрой сервис unit-тестами с моками интерфейсов (100% изоляция от сети и БД).

Stage 5: Transport Layer (REST API & Swagger)

    Реализуй хэндлеры Chi в internal/transport/rest/v1/:

        POST /api/v1/routes/build — основной эндпоинт построения маршрута.

        GET /api/v1/users/{id} — получение профиля.

        POST /api/v1/users — создание/обновление пользователя и его интересов.

    Добавь аннотации Swaggo (@Summary, @Tags, @Param, @Success, @Failure) для всех хэндлеров и DTO.

    Подключи Swagger UI на маршрут /swagger/*.

    Напиши табличные тесты для HTTP-хэндлеров (httptest.ResponseRecorder).

Stage 6: Dependency Injection (uber-go/fx) & Entrypoint

    В internal/app/app.go настрой модули fx.Provide:

        Config, Logger

        Postgres DB Pool, Goose Migrator Runner

        Repositories

        HTTP Clients (Places, ML, Routing)

        Services

        Handlers, Router, HTTP Server Lifecycle (fx.Hook для graceful shutdown)

    В cmd/api/main.go оставь только запуск fx.New(app.Module).Run().

📋 Quality Constraints & Coding Rules

    No Global State: Никаких глобальных переменных для БД или конфигов. Всё прокидывается через конструкторы Fx.

    Context Propagation: Каждый запрос к БД, внешнему API и хэндлеру обязан принимать и пробрасывать context.Context.

    Linter Clean: Код должен соответствовать golangci-lint (обработка всех ошибок, никаких необработанных err).

    Graceful Degradation: Падение внешнего API 2GIS не должно ломать сервис — срабатывает fallback.