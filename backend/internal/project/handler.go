package project

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/internal/middleware"
	"taskflow/internal/models"
	"taskflow/internal/response"
)

type TaskLister interface {
	List(ctx context.Context, projectID uuid.UUID, status, assignee string) ([]models.Task, error)
}

type Handler struct {
	service    *Service
	taskLister TaskLister
}

func NewHandler(service *Service, taskLister TaskLister) *Handler {
	return &Handler{service: service, taskLister: taskLister}
}

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	projects, err := h.service.List(r.Context(), userID)
	if err != nil {
		slog.Error("failed to list projects", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	if projects == nil {
		projects = []models.Project{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := h.service.Create(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			response.ValidationError(w, ve.Fields)
			return
		}
		slog.Error("failed to create project", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusCreated, project)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	project, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		slog.Error("failed to get project", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	tasks, err := h.taskLister.List(r.Context(), id, "", "")
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		slog.Error("failed to list project tasks", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}

	result := models.ProjectWithTasks{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerID:     project.OwnerID,
		CreatedAt:   project.CreatedAt,
		Tasks:       tasks,
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	userID := middleware.GetUserID(r)

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := UpdateProjectInput{}
	validationErrors := map[string]string{}

	if raw, ok := body["name"]; ok {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil || name == "" {
			validationErrors["name"] = "must be a non-empty string"
		} else {
			input.Name = &name
		}
	}

	if raw, ok := body["description"]; ok {
		if string(raw) == "null" {
			input.ClearDescription = true
		} else {
			var desc string
			if err := json.Unmarshal(raw, &desc); err != nil {
				validationErrors["description"] = "must be a string"
			} else {
				input.Description = &desc
			}
		}
	}

	if len(validationErrors) > 0 {
		response.ValidationError(w, validationErrors)
		return
	}

	project, err := h.service.Update(r.Context(), id, userID, input)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		slog.Error("failed to update project", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusOK, project)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	userID := middleware.GetUserID(r)

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		slog.Error("failed to delete project", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
