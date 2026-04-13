package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"taskflow/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scanProject(row interface{ Scan(dest ...any) error }) (*models.Project, error) {
	var p models.Project
	var description sql.NullString
	err := row.Scan(&p.ID, &p.Name, &description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	if description.Valid {
		p.Description = &description.String
	}
	return &p, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		WHERE p.owner_id = $1
		   OR p.id IN (SELECT DISTINCT project_id FROM tasks WHERE assignee_id = $1)
		ORDER BY p.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

func (r *Repository) Create(ctx context.Context, p *models.Project) error {
	query := `INSERT INTO projects (name, description, owner_id)
	          VALUES ($1, $2, $3)
	          RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query, p.Name, p.Description, p.OwnerID).
		Scan(&p.ID, &p.CreatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	query := `SELECT id, name, description, owner_id, created_at FROM projects WHERE id = $1`

	p, err := scanProject(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, input UpdateProjectInput) (*models.Project, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if input.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *input.Name)
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

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE projects SET %s WHERE id = $%d
		 RETURNING id, name, description, owner_id, created_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	p, err := scanProject(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`
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
