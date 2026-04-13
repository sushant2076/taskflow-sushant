package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"taskflow/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (name, email, password)
	          VALUES ($1, $2, $3)
	          RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.Password).
		Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &models.ValidationError{
				Fields: map[string]string{"email": "already registered"},
			}
		}
		return err
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, name, email, password, created_at FROM users WHERE email = $1`

	var u models.User
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
