package task

import (
	"context"
	"time"

	"github.com/google/uuid"

	"taskflow/internal/models"
)

type UpdateTaskInput struct {
	Title            *string
	Description      *string
	ClearDescription bool
	Status           *string
	Priority         *string
	AssigneeID       *uuid.UUID
	ClearAssignee    bool
	DueDate          *string
	ClearDueDate     bool
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID, status, assignee string, limit, offset int) ([]models.Task, int, error) {
	exists, err := s.repo.ProjectExists(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, models.ErrNotFound
	}
	return s.repo.List(ctx, projectID, status, assignee, limit, offset)
}

func (s *Service) Stats(ctx context.Context, projectID uuid.UUID) (*models.ProjectStats, error) {
	exists, err := s.repo.ProjectExists(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.ErrNotFound
	}

	byStatus, err := s.repo.CountByStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}

	byAssignee, err := s.repo.CountByAssignee(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if byAssignee == nil {
		byAssignee = []models.AssigneeStats{}
	}

	total := 0
	for _, c := range byStatus {
		total += c
	}

	return &models.ProjectStats{
		TotalTasks: total,
		ByStatus:   byStatus,
		ByAssignee: byAssignee,
	}, nil
}

func (s *Service) Create(ctx context.Context, projectID, userID uuid.UUID, title string, description *string, status, priority *string, assigneeID *uuid.UUID, dueDate *string) (*models.Task, error) {
	fields := map[string]string{}
	if title == "" {
		fields["title"] = "is required"
	}

	taskStatus := "todo"
	if status != nil {
		if !isValidStatus(*status) {
			fields["status"] = "must be one of: todo, in_progress, done"
		} else {
			taskStatus = *status
		}
	}

	taskPriority := "medium"
	if priority != nil {
		if !isValidPriority(*priority) {
			fields["priority"] = "must be one of: low, medium, high"
		} else {
			taskPriority = *priority
		}
	}

	if dueDate != nil {
		if _, err := time.Parse("2006-01-02", *dueDate); err != nil {
			fields["due_date"] = "must be a valid date (YYYY-MM-DD)"
		}
	}

	if len(fields) > 0 {
		return nil, &models.ValidationError{Fields: fields}
	}

	exists, err := s.repo.ProjectExists(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.ErrNotFound
	}

	t := &models.Task{
		Title:       title,
		Description: description,
		Status:      taskStatus,
		Priority:    taskPriority,
		ProjectID:   projectID,
		AssigneeID:  assigneeID,
		CreatedBy:   userID,
		DueDate:     dueDate,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, taskID uuid.UUID, input UpdateTaskInput) (*models.Task, error) {
	fields := map[string]string{}
	if input.Title != nil && *input.Title == "" {
		fields["title"] = "must be a non-empty string"
	}
	if input.Status != nil && !isValidStatus(*input.Status) {
		fields["status"] = "must be one of: todo, in_progress, done"
	}
	if input.Priority != nil && !isValidPriority(*input.Priority) {
		fields["priority"] = "must be one of: low, medium, high"
	}
	if input.DueDate != nil {
		if _, err := time.Parse("2006-01-02", *input.DueDate); err != nil {
			fields["due_date"] = "must be a valid date (YYYY-MM-DD)"
		}
	}
	if len(fields) > 0 {
		return nil, &models.ValidationError{Fields: fields}
	}

	return s.repo.Update(ctx, taskID, input)
}

func (s *Service) Delete(ctx context.Context, taskID, userID uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	ownerID, err := s.repo.GetProjectOwnerID(ctx, task.ProjectID)
	if err != nil {
		return err
	}

	if userID != ownerID && userID != task.CreatedBy {
		return models.ErrForbidden
	}

	return s.repo.Delete(ctx, taskID)
}

func isValidStatus(s string) bool {
	return s == "todo" || s == "in_progress" || s == "done"
}

func isValidPriority(p string) bool {
	return p == "low" || p == "medium" || p == "high"
}
