package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"taskflow/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scanTask(row interface{ Scan(dest ...any) error }) (*models.Task, error) {
	var t models.Task
	var description sql.NullString
	var assigneeID uuid.NullUUID
	var dueDate sql.NullTime

	err := row.Scan(
		&t.ID, &t.Title, &description, &t.Status, &t.Priority,
		&t.ProjectID, &assigneeID, &t.CreatedBy, &dueDate,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if description.Valid {
		t.Description = &description.String
	}
	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.UUID
	}
	if dueDate.Valid {
		formatted := dueDate.Time.Format("2006-01-02")
		t.DueDate = &formatted
	}
	return &t, nil
}

func (r *Repository) List(ctx context.Context, projectID uuid.UUID, status, assignee string, limit, offset int) ([]models.Task, int, error) {
	whereClause := "WHERE project_id = $1"
	args := []any{projectID}
	argIdx := 2

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if assignee != "" {
		assigneeID, err := uuid.Parse(assignee)
		if err == nil {
			whereClause += fmt.Sprintf(" AND assignee_id = $%d", argIdx)
			args = append(args, assigneeID)
			argIdx++
		}
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(
		`SELECT id, title, description, status, priority, project_id,
		        assignee_id, created_by, due_date, created_at, updated_at
		 FROM tasks %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, total, rows.Err()
}

func (r *Repository) Create(ctx context.Context, t *models.Task) error {
	query := `INSERT INTO tasks (title, description, status, priority, project_id, assignee_id, created_by, due_date)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		t.Title, t.Description, t.Status, t.Priority,
		t.ProjectID, t.AssigneeID, t.CreatedBy, t.DueDate,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return &models.ValidationError{
				Fields: map[string]string{"assignee_id": "user not found"},
			}
		}
		return err
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	query := `SELECT id, title, description, status, priority, project_id,
	                 assignee_id, created_by, due_date, created_at, updated_at
	          FROM tasks WHERE id = $1`

	t, err := scanTask(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, input UpdateTaskInput) (*models.Task, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if input.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *input.Title)
		argIdx++
	}
	if input.ClearDescription {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	} else if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *input.Status)
		argIdx++
	}
	if input.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *input.Priority)
		argIdx++
	}
	if input.ClearAssignee {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	} else if input.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *input.AssigneeID)
		argIdx++
	}
	if input.ClearDueDate {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, nil)
		argIdx++
	} else if input.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *input.DueDate)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE tasks SET %s WHERE id = $%d
		 RETURNING id, title, description, status, priority, project_id,
		           assignee_id, created_by, due_date, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	t, err := scanTask(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, &models.ValidationError{
				Fields: map[string]string{"assignee_id": "user not found"},
			}
		}
		return nil, err
	}
	return t, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repository) CountByStatus(ctx context.Context, projectID uuid.UUID) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM tasks WHERE project_id = $1 GROUP BY status`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *Repository) CountByAssignee(ctx context.Context, projectID uuid.UUID) ([]models.AssigneeStats, error) {
	query := `SELECT t.assignee_id, u.name, COUNT(*)
	          FROM tasks t
	          LEFT JOIN users u ON u.id = t.assignee_id
	          WHERE t.project_id = $1
	          GROUP BY t.assignee_id, u.name
	          ORDER BY COUNT(*) DESC`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.AssigneeStats
	for rows.Next() {
		var s models.AssigneeStats
		var assigneeID uuid.NullUUID
		var name sql.NullString
		if err := rows.Scan(&assigneeID, &name, &s.Count); err != nil {
			return nil, err
		}
		if assigneeID.Valid {
			s.AssigneeID = &assigneeID.UUID
		}
		if name.Valid {
			s.Name = &name.String
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *Repository) ProjectExists(ctx context.Context, projectID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)", projectID,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) GetProjectOwnerID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	var ownerID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		"SELECT owner_id FROM projects WHERE id = $1", projectID,
	).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, models.ErrNotFound
		}
		return uuid.Nil, err
	}
	return ownerID, nil
}
