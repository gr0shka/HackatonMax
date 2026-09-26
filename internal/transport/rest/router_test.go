package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
	"HackatonMax/internal/transport/rest"
	v1 "HackatonMax/internal/transport/rest/v1"
)

type mockUserSvc struct {
	mock.Mock
}

func (m *mockUserSvc) GetUser(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserSvc) CreateOrUpdateUser(ctx context.Context, name, email string, interests entity.Interests) (*entity.User, error) {
	args := m.Called(ctx, name, email, interests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserSvc) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *mockUserSvc) GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.User), args.Error(1)
}

type mockRouteSvc struct {
	mock.Mock
}

func (m *mockRouteSvc) BuildRoute(ctx context.Context, params service.BuildRouteParams) (*entity.Route, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Route), args.Error(1)
}

func TestRouter_HealthAndSwagger(t *testing.T) {
	userSvc := new(mockUserSvc)
	routeSvc := new(mockRouteSvc)

	v1Handler := v1.NewHandler(routeSvc, userSvc)
	router := rest.NewRouter(v1Handler, rest.RouterConfig{})

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")

	// 1b. API v1 Health check
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")

	// 2. Swagger UI index
	req = httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "swagger")

	// 3. Swagger doc.json
	req = httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "API Оптимизации Туристических Маршрутов")
	assert.Contains(t, w.Body.String(), "1.0.0")
}
