package auth

import (
	"context"
	"net/mail"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"taskflow/internal/models"
)

type Service struct {
	repo       *Repository
	jwtSecret  string
	bcryptCost int
}

func NewService(repo *Repository, jwtSecret string, bcryptCost int) *Service {
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		bcryptCost: bcryptCost,
	}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (*models.User, string, error) {
	fields := map[string]string{}
	if name == "" {
		fields["name"] = "is required"
	}
	if email == "" {
		fields["email"] = "is required"
	} else if _, err := mail.ParseAddress(email); err != nil {
		fields["email"] = "is not a valid email"
	}
	if password == "" {
		fields["password"] = "is required"
	} else if len(password) < 6 {
		fields["password"] = "must be at least 6 characters"
	}
	if len(fields) > 0 {
		return nil, "", &models.ValidationError{Fields: fields}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, "", err
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	fields := map[string]string{}
	if email == "" {
		fields["email"] = "is required"
	}
	if password == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		return nil, "", &models.ValidationError{Fields: fields}
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", models.ErrNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", models.ErrNotFound
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *Service) generateToken(user *models.User) (string, error) {
	claims := models.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
