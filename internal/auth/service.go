package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bayarin/backend/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegisterOwnerInput holds data to create a new business + owner.
type RegisterOwnerInput struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	OwnerName    string `json:"owner_name"`
}

// LoginInput for sign-in.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse is the user payload returned to clients.
type UserResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	BusinessID uuid.UUID  `json:"business_id"`
	BranchID   *uuid.UUID `json:"branch_id,omitempty"`
}

// LoginResponse bundles the token and user together.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// Service handles authentication business logic.
type Service struct {
	db *sql.DB
}

// NewService creates a new auth service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// RegisterOwner atomically creates the business and owner user.
func (s *Service) RegisterOwner(input RegisterOwnerInput) (*UserResponse, error) {
	if input.Email == "" || input.Password == "" || input.BusinessName == "" || input.OwnerName == "" {
		return nil, errors.New("all fields are required")
	}
	if len(input.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	slug := slugify(input.BusinessName)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var businessID uuid.UUID
	err = tx.QueryRow(
		`INSERT INTO businesses (name, slug) VALUES ($1, $2) RETURNING id`,
		input.BusinessName, slug,
	).Scan(&businessID)
	if err != nil {
		return nil, fmt.Errorf("insert business: %w", err)
	}

	var userID uuid.UUID
	err = tx.QueryRow(
		`INSERT INTO users (business_id, name, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, 'owner') RETURNING id`,
		businessID, input.OwnerName, strings.ToLower(input.Email), string(hash),
	).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &UserResponse{
		ID:         userID,
		Name:       input.OwnerName,
		Email:      strings.ToLower(input.Email),
		Role:       "owner",
		BusinessID: businessID,
	}, nil
}

// Login authenticates a user and issues a session token.
func (s *Service) Login(input LoginInput) (*LoginResponse, error) {
	var (
		id           uuid.UUID
		name, role   string
		passwordHash string
		businessID   uuid.UUID
		branchID     sql.NullString
		isActive     bool
	)

	row := s.db.QueryRow(
		`SELECT id, name, role, password_hash, business_id, branch_id, is_active
		 FROM users WHERE email = $1`,
		strings.ToLower(input.Email),
	)
	if err := row.Scan(&id, &name, &role, &passwordHash, &businessID, &branchID, &isActive); err != nil {
		return nil, errors.New("invalid credentials")
	}
	if !isActive {
		return nil, errors.New("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":         id.String(),
		"business_id": businessID.String(),
		"role":        role,
		"exp":         expiresAt.Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := jwtToken.SignedString([]byte(config.App.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	// Persist session.
	_, err = s.db.Exec(
		`INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		id, tokenStr, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	resp := &LoginResponse{
		Token: tokenStr,
		User: UserResponse{
			ID:         id,
			Name:       name,
			Email:      input.Email,
			Role:       role,
			BusinessID: businessID,
		},
	}
	if branchID.Valid {
		parsed, _ := uuid.Parse(branchID.String)
		resp.User.BranchID = &parsed
	}
	return resp, nil
}

// Logout revokes the session associated with the given token.
func (s *Service) Logout(token string) error {
	_, err := s.db.Exec(`UPDATE sessions SET revoked = true WHERE token = $1`, token)
	return err
}

// Me fetches the user record for the given userID.
func (s *Service) Me(userID uuid.UUID) (*UserResponse, error) {
	var (
		name, email, role string
		businessID        uuid.UUID
		branchID          sql.NullString
	)
	row := s.db.QueryRow(
		`SELECT name, email, role, business_id, branch_id FROM users WHERE id = $1`,
		userID,
	)
	if err := row.Scan(&name, &email, &role, &businessID, &branchID); err != nil {
		return nil, errors.New("user not found")
	}
	resp := &UserResponse{
		ID:         userID,
		Name:       name,
		Email:      email,
		Role:       role,
		BusinessID: businessID,
	}
	if branchID.Valid {
		parsed, _ := uuid.Parse(branchID.String)
		resp.BranchID = &parsed
	}
	return resp, nil
}

// slugify converts a name to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric except dashes.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
