package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProjectWithTasks struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	Tasks       []Task    `json:"tasks"`
}

type Task struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	ProjectID   uuid.UUID  `json:"project_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	DueDate     *string    `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ProjectStats struct {
	TotalTasks int              `json:"total_tasks"`
	ByStatus   map[string]int   `json:"by_status"`
	ByAssignee []AssigneeStats  `json:"by_assignee"`
}

type AssigneeStats struct {
	AssigneeID *uuid.UUID `json:"assignee_id"`
	Name       *string    `json:"name"`
	Count      int        `json:"count"`
}

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}
