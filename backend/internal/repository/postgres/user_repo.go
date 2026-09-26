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

// UserRepository implements service.UserRepository with PostgreSQL backend.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

// GetUserByID retrieves a user profile by their unique ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, name, email, interests, created_at, updated_at
		FROM users
		WHERE id = $1;
	`

	var user entity.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Interests,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id %s: %w", id, err)
	}

	return &user, nil
}

// GetUsersByIDs retrieves multiple user profiles by their IDs (e.g. group members or friends).
func (r *UserRepository) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error) {
	if len(ids) == 0 {
		return []entity.User{}, nil
	}

	query := `
		SELECT id, name, email, interests, created_at, updated_at
		FROM users
		WHERE id = ANY($1);
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by ids: %w", err)
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Interests,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users rows: %w", err)
	}

	return users, nil
}

// CreateUser inserts a new user profile into the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	query := `
		INSERT INTO users (id, name, email, interests, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at;
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.Interests,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user's name, email, and interests.
func (r *UserRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users
		SET name = $2, email = $3, interests = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at;
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.Interests,
	).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ErrUserNotFound
		}
		return fmt.Errorf("failed to update user %s: %w", user.ID, err)
	}

	return nil
}

// AddFriend creates an association link between two users.
func (r *UserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return fmt.Errorf("user cannot be friends with themselves")
	}

	query := `
		INSERT INTO user_friends (user_id, friend_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, friend_id) DO NOTHING;
	`

	_, err := r.pool.Exec(ctx, query, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to add friend relationship: %w", err)
	}

	return nil
}

// GetFriends returns all friends associated with the specified user.
func (r *UserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.interests, u.created_at, u.updated_at
		FROM users u
		JOIN user_friends uf ON u.id = uf.friend_id
		WHERE uf.user_id = $1
		ORDER BY u.name ASC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query friends for user %s: %w", userID, err)
	}
	defer rows.Close()

	var friends []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Interests,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan friend: %w", err)
		}
		friends = append(friends, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating friends rows: %w", err)
	}

	return friends, nil
}
