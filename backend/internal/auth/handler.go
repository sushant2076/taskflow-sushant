package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"taskflow/internal/models"
	"taskflow/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			response.ValidationError(w, ve.Fields)
			return
		}
		slog.Error("failed to register user", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			response.ValidationError(w, ve.Fields)
			return
		}
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("failed to login user", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusOK, authResponse{Token: token, User: user})
}
