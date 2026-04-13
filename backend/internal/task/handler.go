package task

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/internal/middleware"
	"taskflow/internal/models"
	"taskflow/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	DueDate     *string    `json:"due_date"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	status := r.URL.Query().Get("status")
	assignee := r.URL.Query().Get("assignee")

	tasks, err := h.service.List(r.Context(), projectID, status, assignee)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		slog.Error("failed to list tasks", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	userID := middleware.GetUserID(r)

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.service.Create(r.Context(), projectID, userID,
		req.Title, req.Description, req.Status, req.Priority,
		req.AssigneeID, req.DueDate,
	)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			response.ValidationError(w, ve.Fields)
			return
		}
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		slog.Error("failed to create task", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusCreated, task)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := UpdateTaskInput{}
	validationErrors := map[string]string{}

	if raw, ok := body["title"]; ok {
		var title string
		if err := json.Unmarshal(raw, &title); err != nil || title == "" {
			validationErrors["title"] = "must be a non-empty string"
		} else {
			input.Title = &title
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

	if raw, ok := body["status"]; ok {
		var status string
		if err := json.Unmarshal(raw, &status); err != nil {
			validationErrors["status"] = "must be a string"
		} else {
			input.Status = &status
		}
	}

	if raw, ok := body["priority"]; ok {
		var priority string
		if err := json.Unmarshal(raw, &priority); err != nil {
			validationErrors["priority"] = "must be a string"
		} else {
			input.Priority = &priority
		}
	}

	if raw, ok := body["assignee_id"]; ok {
		if string(raw) == "null" {
			input.ClearAssignee = true
		} else {
			var idStr string
			if err := json.Unmarshal(raw, &idStr); err != nil {
				validationErrors["assignee_id"] = "must be a valid UUID"
			} else {
				parsed, err := uuid.Parse(idStr)
				if err != nil {
					validationErrors["assignee_id"] = "must be a valid UUID"
				} else {
					input.AssigneeID = &parsed
				}
			}
		}
	}

	if raw, ok := body["due_date"]; ok {
		if string(raw) == "null" {
			input.ClearDueDate = true
		} else {
			var dateStr string
			if err := json.Unmarshal(raw, &dateStr); err != nil {
				validationErrors["due_date"] = "must be a date string (YYYY-MM-DD)"
			} else if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				validationErrors["due_date"] = "must be a valid date (YYYY-MM-DD)"
			} else {
				input.DueDate = &dateStr
			}
		}
	}

	if len(validationErrors) > 0 {
		response.ValidationError(w, validationErrors)
		return
	}

	task, err := h.service.Update(r.Context(), taskID, input)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			response.ValidationError(w, ve.Fields)
			return
		}
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		slog.Error("failed to update task", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.JSON(w, http.StatusOK, task)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	userID := middleware.GetUserID(r)

	if err := h.service.Delete(r.Context(), taskID, userID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "not found")
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		slog.Error("failed to delete task", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
