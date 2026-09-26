package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"HackatonMax/internal/entity"
)

type userService struct {
	userRepo UserRepository
}

// NewUserService creates an implementation of UserService.
func NewUserService(userRepo UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user id", ErrUserNotFound)
	}
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *userService) CreateOrUpdateUser(ctx context.Context, name, email string, interests entity.Interests) (*entity.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name == "" {
		return nil, fmt.Errorf("user name cannot be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("user email cannot be empty")
	}

	if interests == nil {
		interests = make(entity.Interests)
	}

	user := &entity.User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Interests: interests,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

func (s *userService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == uuid.Nil || friendID == uuid.Nil {
		return fmt.Errorf("user ids cannot be empty")
	}
	if userID == friendID {
		return fmt.Errorf("user cannot be friends with themselves")
	}

	// Verify both users exist
	if _, err := s.userRepo.GetUserByID(ctx, userID); err != nil {
		return fmt.Errorf("user %s not found: %w", userID, err)
	}
	if _, err := s.userRepo.GetUserByID(ctx, friendID); err != nil {
		return fmt.Errorf("friend %s not found: %w", friendID, err)
	}

	return s.userRepo.AddFriend(ctx, userID, friendID)
}

func (s *userService) GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user id", ErrUserNotFound)
	}
	return s.userRepo.GetFriends(ctx, userID)
}
