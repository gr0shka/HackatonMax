package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"HackatonMax/internal/service"
	"HackatonMax/internal/transport/rest/v1/dto"
)

// GetUser godoc
// @Summary      Get user profile
// @Description  Retrieves profile information and interest weights for a user by UUID
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  dto.UserResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/users/{id} [get]
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id format", "must be a valid UUID")
		return
	}

	user, err := h.userService.GetUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "user not found", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get user", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		Interests: user.Interests,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// CreateOrUpdateUser godoc
// @Summary      Create or update user profile
// @Description  Creates a new user profile or updates interests
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateOrUpdateUserRequest true "User data"
// @Success      200  {object}  dto.UserResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/users [post]
func (h *Handler) CreateOrUpdateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateOrUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json payload", err.Error())
		return
	}

	if req.Name == "" || req.Email == "" {
		respondError(w, http.StatusBadRequest, "name and email are required", "")
		return
	}

	interests := req.Interests
	if interests == nil {
		interests = make(map[string]float64)
	}

	user, err := h.userService.CreateOrUpdateUser(r.Context(), req.Name, req.Email, interests)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save user", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		Interests: user.Interests,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}
