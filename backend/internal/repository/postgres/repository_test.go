package postgres_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/repository/postgres"
	"HackatonMax/internal/service"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgis/postgis:16-3.4",
		tcpostgres.WithDatabase("test_route_optimizer"),
		tcpostgres.WithUsername("test_user"),
		tcpostgres.WithPassword("test_pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgis testcontainer")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	pool, err := postgres.NewPostgresPool(ctx, connStr)
	require.NoError(t, err, "failed to create postgres connection pool")

	migrationsDir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err, "failed to get absolute migrations path")

	err = postgres.RunMigrations(ctx, pool, migrationsDir)
	require.NoError(t, err, "failed to apply goose migrations on testcontainer")

	cleanup := func() {
		postgres.ClosePostgresPool(pool)
		_ = pgContainer.Terminate(context.Background())
	}

	return pool, cleanup
}

func TestUserRepository(t *testing.T) {
	pool, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()
	userRepo := postgres.NewUserRepository(pool)

	user1 := &entity.User{
		ID:    uuid.New(),
		Name:  "Алексей",
		Email: "alexey@example.com",
		Interests: entity.Interests{
			"culture": 0.9,
			"cafe":    0.6,
		},
	}

	user2 := &entity.User{
		ID:    uuid.New(),
		Name:  "Екатерина",
		Email: "kate@example.com",
		Interests: entity.Interests{
			"park": 0.8,
			"cafe": 0.7,
		},
	}

	// 1. CreateUser
	err := userRepo.CreateUser(ctx, user1)
	require.NoError(t, err)
	assert.False(t, user1.CreatedAt.IsZero())

	err = userRepo.CreateUser(ctx, user2)
	require.NoError(t, err)

	// 2. GetUserByID
	fetched1, err := userRepo.GetUserByID(ctx, user1.ID)
	require.NoError(t, err)
	assert.Equal(t, user1.Name, fetched1.Name)
	assert.Equal(t, user1.Email, fetched1.Email)
	assert.Equal(t, 0.9, fetched1.Interests["culture"])
	assert.Equal(t, 0.6, fetched1.Interests["cafe"])

	// 3. GetUserByID NotFound
	_, err = userRepo.GetUserByID(ctx, uuid.New())
	assert.ErrorIs(t, err, service.ErrUserNotFound)

	// 4. UpdateUser
	user1.Name = "Алексей Обновленный"
	user1.Interests["museum"] = 0.95
	err = userRepo.UpdateUser(ctx, user1)
	require.NoError(t, err)

	updatedUser1, err := userRepo.GetUserByID(ctx, user1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Алексей Обновленный", updatedUser1.Name)
	assert.Equal(t, 0.95, updatedUser1.Interests["museum"])

	// 5. AddFriend and GetFriends
	err = userRepo.AddFriend(ctx, user1.ID, user2.ID)
	require.NoError(t, err)

	friends, err := userRepo.GetFriends(ctx, user1.ID)
	require.NoError(t, err)
	require.Len(t, friends, 1)
	assert.Equal(t, user2.ID, friends[0].ID)
	assert.Equal(t, user2.Name, friends[0].Name)

	// 6. GetUsersByIDs
	users, err := userRepo.GetUsersByIDs(ctx, []uuid.UUID{user1.ID, user2.ID})
	require.NoError(t, err)
	assert.Len(t, users, 2)

	// Empty slice of IDs
	empty, err := userRepo.GetUsersByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestPlaceRepository_SpatialAndCRUD(t *testing.T) {
	pool, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()
	placeRepo := postgres.NewPlaceRepository(pool)

	// Coordinates around Moscow Center:
	// Red Square: (55.7539, 37.6208)
	// Bolshoi Theatre: (55.7601, 37.6186) ~700m from Red Square
	// Gorky Park: (55.7296, 37.6015) ~3000m from Red Square
	// Sokolniki Park: (55.7936, 37.6775) ~6000m from Red Square
	places := []entity.Place{
		{
			ID:             uuid.New(),
			ExternalID:     "place-red-square",
			Name:           "Красная Площадь",
			Address:        "Красная пл.",
			Category:       "attraction",
			Rating:         4.9,
			AvgDurationMin: 60,
			Lat:            55.7539,
			Lon:            37.6208,
		},
		{
			ID:             uuid.New(),
			ExternalID:     "place-bolshoi",
			Name:           "Большой Театр",
			Address:        "Театральная пл., 1",
			Category:       "theatre",
			Rating:         4.8,
			AvgDurationMin: 120,
			Lat:            55.7601,
			Lon:            37.6186,
		},
		{
			ID:             uuid.New(),
			ExternalID:     "place-gorky-park",
			Name:           "Парк Горького",
			Address:        "Крымский Вал, 9",
			Category:       "park",
			Rating:         4.7,
			AvgDurationMin: 90,
			Lat:            55.7296,
			Lon:            37.6015,
		},
		{
			ID:             uuid.New(),
			ExternalID:     "place-sokolniki",
			Name:           "Парк Сокольники",
			Address:        "Сокольнический Вал, 1",
			Category:       "park",
			Rating:         4.6,
			AvgDurationMin: 90,
			Lat:            55.7936,
			Lon:            37.6775,
		},
	}

	// 1. Bulk insert via SavePlaces
	err := placeRepo.SavePlaces(ctx, places)
	require.NoError(t, err)

	// 2. GetPlaceByID & GetPlaceByExternalID
	p1, err := placeRepo.GetPlaceByID(ctx, places[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "Красная Площадь", p1.Name)
	assert.Equal(t, 55.7539, p1.Lat)
	assert.Equal(t, 37.6208, p1.Lon)

	pExt, err := placeRepo.GetPlaceByExternalID(ctx, "place-bolshoi")
	require.NoError(t, err)
	assert.Equal(t, "Большой Театр", pExt.Name)

	// Not found
	_, err = placeRepo.GetPlaceByID(ctx, uuid.New())
	assert.ErrorIs(t, err, service.ErrPlaceNotFound)

	// 3. FindNearbyPlaces within 1000m of Red Square (55.7539, 37.6208)
	// Should find Red Square (0m) and Bolshoi (~700m), but NOT Gorky Park (~3000m) or Sokolniki (~6000m)
	nearby1km, err := placeRepo.FindNearbyPlaces(ctx, 55.7539, 37.6208, 1000, 10)
	require.NoError(t, err)
	require.Len(t, nearby1km, 2)
	assert.Equal(t, "Красная Площадь", nearby1km[0].Name)
	assert.Equal(t, "Большой Театр", nearby1km[1].Name)

	// 4. FindNearbyPlaces within 4000m of Red Square
	// Should include Red Square, Bolshoi, and Gorky Park (3 places)
	nearby4km, err := placeRepo.FindNearbyPlaces(ctx, 55.7539, 37.6208, 4000, 10)
	require.NoError(t, err)
	require.Len(t, nearby4km, 3)

	// 5. FindNearbyPlacesByCategory within 4000m for "park"
	parks, err := placeRepo.FindNearbyPlacesByCategory(ctx, 55.7539, 37.6208, 4000, "park", 10)
	require.NoError(t, err)
	require.Len(t, parks, 1)
	assert.Equal(t, "Парк Горького", parks[0].Name)

	// 6. Test Upsert via SavePlace (update rating and name)
	updatedBolshoi := *pExt
	updatedBolshoi.Name = "Большой Театр России"
	updatedBolshoi.Rating = 5.0
	err = placeRepo.SavePlace(ctx, &updatedBolshoi)
	require.NoError(t, err)

	refetched, err := placeRepo.GetPlaceByExternalID(ctx, "place-bolshoi")
	require.NoError(t, err)
	assert.Equal(t, "Большой Театр России", refetched.Name)
	assert.Equal(t, 5.0, refetched.Rating)
}
