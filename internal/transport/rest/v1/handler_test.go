package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"HackatonMax/internal/entity"
	"HackatonMax/internal/service"
	v1 "HackatonMax/internal/transport/rest/v1"
	"HackatonMax/internal/transport/rest/v1/dto"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetUser(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserService) CreateOrUpdateUser(ctx context.Context, name, email string, interests entity.Interests) (*entity.User, error) {
	args := m.Called(ctx, name, email, interests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserService) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *mockUserService) GetFriends(ctx context.Context, userID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.User), args.Error(1)
}

type mockRouteService struct {
	mock.Mock
}

func (m *mockRouteService) BuildRoute(ctx context.Context, params service.BuildRouteParams) (*entity.Route, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Route), args.Error(1)
}

func TestHandler_GetUser_TableDriven(t *testing.T) {
	validID := uuid.New()
	existingUser := &entity.User{
		ID:        validID,
		Name:      "Алексей",
		Email:     "alex@example.com",
		Interests: entity.Interests{"coffee": 0.8},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		paramID        string
		setupMock      func(m *mockUserService)
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:    "success 200",
			paramID: validID.String(),
			setupMock: func(m *mockUserService) {
				m.On("GetUser", mock.Anything, validID).Return(existingUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.UserResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, validID.String(), resp.ID)
				assert.Equal(t, "Алексей", resp.Name)
				assert.Equal(t, 0.8, resp.Interests["coffee"])
			},
		},
		{
			name:    "not found 404",
			paramID: validID.String(),
			setupMock: func(m *mockUserService) {
				m.On("GetUser", mock.Anything, validID).Return(nil, service.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var errResp dto.ErrorResponse
				err := json.NewDecoder(w.Body).Decode(&errResp)
				require.NoError(t, err)
				assert.Equal(t, "user not found", errResp.Error)
			},
		},
		{
			name:           "invalid uuid 400",
			paramID:        "invalid-not-uuid",
			setupMock:      func(m *mockUserService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var errResp dto.ErrorResponse
				err := json.NewDecoder(w.Body).Decode(&errResp)
				require.NoError(t, err)
				assert.Equal(t, "invalid user id format", errResp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userSvc := new(mockUserService)
			routeSvc := new(mockRouteService)
			tt.setupMock(userSvc)

			handler := v1.NewHandler(routeSvc, userSvc)

			r := chi.NewRouter()
			r.Get("/users/{id}", handler.GetUser)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.paramID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandler_CreateOrUpdateUser_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(m *mockUserService)
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "success 200",
			body: `{"name":"Михаил","email":"mikhail@example.com","interests":{"park":0.9}}`,
			setupMock: func(m *mockUserService) {
				m.On("CreateOrUpdateUser", mock.Anything, "Михаил", "mikhail@example.com", entity.Interests{"park": 0.9}).
					Return(&entity.User{
						ID:        uuid.New(),
						Name:      "Михаил",
						Email:     "mikhail@example.com",
						Interests: entity.Interests{"park": 0.9},
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.UserResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, "Михаил", resp.Name)
				assert.Equal(t, "mikhail@example.com", resp.Email)
			},
		},
		{
			name:           "invalid json 400",
			body:           `{not valid json`,
			setupMock:      func(m *mockUserService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "invalid json payload")
			},
		},
		{
			name:           "missing required fields 400",
			body:           `{"name":"","email":""}`,
			setupMock:      func(m *mockUserService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "name and email are required")
			},
		},
		{
			name: "internal error 500",
			body: `{"name":"Михаил","email":"mikhail@example.com"}`,
			setupMock: func(m *mockUserService) {
				m.On("CreateOrUpdateUser", mock.Anything, "Михаил", "mikhail@example.com", entity.Interests{}).
					Return(nil, errors.New("db down")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "failed to save user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userSvc := new(mockUserService)
			routeSvc := new(mockRouteService)
			tt.setupMock(userSvc)

			handler := v1.NewHandler(routeSvc, userSvc)

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.CreateOrUpdateUser(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandler_BuildRoute_TableDriven(t *testing.T) {
	uID := uuid.New()
	sampleRoute := &entity.Route{
		ID:                  uuid.New(),
		MatchScore:          0.91,
		MatchReasons:        []string{"Отличный баланс"},
		TotalDurationMin:    90,
		TotalDistanceMeters: 2500,
		Points: []entity.RoutePoint{
			{Order: 0, Type: entity.RoutePointTypeStart, Location: entity.LatLon{Lat: 55.75, Lon: 37.61}},
			{Order: 1, Type: entity.RoutePointTypeFinish, Location: entity.LatLon{Lat: 55.76, Lon: 37.62}},
		},
	}

	tests := []struct {
		name           string
		body           string
		setupMock      func(m *mockRouteService)
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "success 200",
			body: `{"start":{"lat":55.75,"lon":37.61},"finish":{"lat":55.76,"lon":37.62},"budget_minutes":90,"user_ids":["` + uID.String() + `"]}`,
			setupMock: func(m *mockRouteService) {
				m.On("BuildRoute", mock.Anything, mock.MatchedBy(func(p service.BuildRouteParams) bool {
					return p.BudgetMinutes == 90 && len(p.UserIDs) == 1 && p.UserIDs[0] == uID
				})).Return(sampleRoute, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.BuildRouteResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, 0.91, resp.MatchScore)
				assert.Equal(t, 90, resp.TotalDurationMin)
				assert.Equal(t, 2500.0, resp.TotalDistanceMeters)
				assert.Len(t, resp.Waypoints, 2)
			},
		},
		{
			name:           "empty user_ids 400",
			body:           `{"start":{"lat":55.75,"lon":37.61},"finish":{"lat":55.76,"lon":37.62},"budget_minutes":90,"user_ids":[]}`,
			setupMock:      func(m *mockRouteService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "user_ids must contain at least one user")
			},
		},
		{
			name:           "invalid user_id uuid 400",
			body:           `{"start":{"lat":55.75,"lon":37.61},"finish":{"lat":55.76,"lon":37.62},"budget_minutes":90,"user_ids":["not-uuid"]}`,
			setupMock:      func(m *mockRouteService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "invalid user id in list")
			},
		},
		{
			name: "user not found 404",
			body: `{"start":{"lat":55.75,"lon":37.61},"finish":{"lat":55.76,"lon":37.62},"budget_minutes":90,"user_ids":["` + uID.String() + `"]}`,
			setupMock: func(m *mockRouteService) {
				m.On("BuildRoute", mock.Anything, mock.Anything).Return(nil, service.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "user profile not found")
			},
		},
		{
			name: "no places found 404",
			body: `{"start":{"lat":55.75,"lon":37.61},"finish":{"lat":55.76,"lon":37.62},"budget_minutes":90,"user_ids":["` + uID.String() + `"]}`,
			setupMock: func(m *mockRouteService) {
				m.On("BuildRoute", mock.Anything, mock.Anything).Return(nil, service.ErrNoPlacesFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "no places found for route")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userSvc := new(mockUserService)
			routeSvc := new(mockRouteService)
			tt.setupMock(routeSvc)

			handler := v1.NewHandler(routeSvc, userSvc)

			req := httptest.NewRequest(http.MethodPost, "/routes/build", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.BuildRoute(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}
