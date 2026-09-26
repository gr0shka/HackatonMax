package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
)

// PlaceRepository implements service.PlaceRepository with PostGIS spatial support.
type PlaceRepository struct {
	pool *pgxpool.Pool
}

// NewPlaceRepository creates a new PlaceRepository instance.
func NewPlaceRepository(pool *pgxpool.Pool) *PlaceRepository {
	return &PlaceRepository{
		pool: pool,
	}
}

// FindNearbyPlaces retrieves places within radiusMeters from coordinates (lat, lon)
// using PostGIS ST_DWithin and ordered by distance ascending.
func (r *PlaceRepository) FindNearbyPlaces(ctx context.Context, lat, lon float64, radiusMeters float64, limit int) ([]entity.Place, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at
		FROM places
		WHERE ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC
		LIMIT $4;
	`

	rows, err := r.pool.Query(ctx, query, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute FindNearbyPlaces: %w", err)
	}
	defer rows.Close()

	var places []entity.Place
	for rows.Next() {
		var p entity.Place
		if err := rows.Scan(
			&p.ID,
			&p.ExternalID,
			&p.Name,
			&p.Address,
			&p.Category,
			&p.Rating,
			&p.AvgDurationMin,
			&p.Lat,
			&p.Lon,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan place: %w", err)
		}
		places = append(places, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nearby places: %w", err)
	}

	return places, nil
}

// FindNearbyPlacesByCategory retrieves nearby places filtered by category within radiusMeters.
func (r *PlaceRepository) FindNearbyPlacesByCategory(ctx context.Context, lat, lon float64, radiusMeters float64, category string, limit int) ([]entity.Place, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at
		FROM places
		WHERE category = $1
		  AND ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4)
		ORDER BY ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) ASC
		LIMIT $5;
	`

	rows, err := r.pool.Query(ctx, query, category, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute FindNearbyPlacesByCategory: %w", err)
	}
	defer rows.Close()

	var places []entity.Place
	for rows.Next() {
		var p entity.Place
		if err := rows.Scan(
			&p.ID,
			&p.ExternalID,
			&p.Name,
			&p.Address,
			&p.Category,
			&p.Rating,
			&p.AvgDurationMin,
			&p.Lat,
			&p.Lon,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan place: %w", err)
		}
		places = append(places, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nearby places: %w", err)
	}

	return places, nil
}

// GetPlaceByID fetches a place by its internal UUID.
func (r *PlaceRepository) GetPlaceByID(ctx context.Context, id uuid.UUID) (*entity.Place, error) {
	query := `
		SELECT id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at
		FROM places
		WHERE id = $1;
	`

	var p entity.Place
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.ExternalID,
		&p.Name,
		&p.Address,
		&p.Category,
		&p.Rating,
		&p.AvgDurationMin,
		&p.Lat,
		&p.Lon,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrPlaceNotFound
		}
		return nil, fmt.Errorf("failed to get place by id %s: %w", id, err)
	}

	return &p, nil
}

// GetPlaceByExternalID fetches a place by its external 2GIS or OSM identifier.
func (r *PlaceRepository) GetPlaceByExternalID(ctx context.Context, externalID string) (*entity.Place, error) {
	query := `
		SELECT id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at
		FROM places
		WHERE external_id = $1;
	`

	var p entity.Place
	err := r.pool.QueryRow(ctx, query, externalID).Scan(
		&p.ID,
		&p.ExternalID,
		&p.Name,
		&p.Address,
		&p.Category,
		&p.Rating,
		&p.AvgDurationMin,
		&p.Lat,
		&p.Lon,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrPlaceNotFound
		}
		return nil, fmt.Errorf("failed to get place by external id %s: %w", externalID, err)
	}

	return &p, nil
}

// SavePlace upserts a place record by external_id.
func (r *PlaceRepository) SavePlace(ctx context.Context, place *entity.Place) error {
	if place.ID == uuid.Nil {
		place.ID = uuid.New()
	}

	query := `
		INSERT INTO places (id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			category = EXCLUDED.category,
			rating = EXCLUDED.rating,
			avg_duration_min = EXCLUDED.avg_duration_min,
			lat = EXCLUDED.lat,
			lon = EXCLUDED.lon,
			updated_at = NOW()
		RETURNING id, created_at, updated_at;
	`

	err := r.pool.QueryRow(ctx, query,
		place.ID,
		place.ExternalID,
		place.Name,
		place.Address,
		place.Category,
		place.Rating,
		place.AvgDurationMin,
		place.Lat,
		place.Lon,
	).Scan(&place.ID, &place.CreatedAt, &place.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save place: %w", err)
	}

	return nil
}

// SavePlaces bulk-upserts a slice of places within a transaction.
func (r *PlaceRepository) SavePlaces(ctx context.Context, places []entity.Place) error {
	if len(places) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		INSERT INTO places (id, external_id, name, address, category, rating, avg_duration_min, lat, lon, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			category = EXCLUDED.category,
			rating = EXCLUDED.rating,
			avg_duration_min = EXCLUDED.avg_duration_min,
			lat = EXCLUDED.lat,
			lon = EXCLUDED.lon,
			updated_at = NOW();
	`

	batch := &pgx.Batch{}
	for i := range places {
		p := &places[i]
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		batch.Queue(query,
			p.ID,
			p.ExternalID,
			p.Name,
			p.Address,
			p.Category,
			p.Rating,
			p.AvgDurationMin,
			p.Lat,
			p.Lon,
		)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < len(places); i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("failed to execute batch item %d: %w", i, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit save places transaction: %w", err)
	}

	return nil
}
