package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
)

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserRepo) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]entity.User), args.Error(1)
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *mockUserRepo) GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.User), args.Error(1)
}

func TestUserService_GetUser(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)

	userID := uuid.New()
	expectedUser := &entity.User{
		ID:    userID,
		Name:  "Иван",
		Email: "ivan@example.com",
	}

	repo.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)

	user, err := svc.GetUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	// Not found
	notFoundID := uuid.New()
	repo.On("GetUserByID", mock.Anything, notFoundID).Return(nil, service.ErrUserNotFound)
	_, err = svc.GetUser(context.Background(), notFoundID)
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestUserService_CreateOrUpdateUser(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)

	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	interests := entity.Interests{"cafe": 0.8}
	user, err := svc.CreateOrUpdateUser(context.Background(), "Анна", "anna@example.com", interests)
	require.NoError(t, err)
	assert.Equal(t, "Анна", user.Name)
	assert.Equal(t, "anna@example.com", user.Email)
	assert.Equal(t, 0.8, user.Interests["cafe"])

	// Empty name validation
	_, err = svc.CreateOrUpdateUser(context.Background(), "", "anna@example.com", nil)
	assert.Error(t, err)
}

func TestUserService_AddFriend(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)

	u1 := uuid.New()
	u2 := uuid.New()

	repo.On("GetUserByID", mock.Anything, u1).Return(&entity.User{ID: u1}, nil)
	repo.On("GetUserByID", mock.Anything, u2).Return(&entity.User{ID: u2}, nil)
	repo.On("AddFriend", mock.Anything, u1, u2).Return(nil)

	err := svc.AddFriend(context.Background(), u1, u2)
	require.NoError(t, err)

	// Self-friend error
	err = svc.AddFriend(context.Background(), u1, u1)
	assert.Error(t, err)
}
