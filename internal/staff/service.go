package staff

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Staff represents a user (staff member).
type Staff struct {
	ID         uuid.UUID  `json:"id"`
	BusinessID uuid.UUID  `json:"business_id"`
	BranchID   *uuid.UUID `json:"branch_id,omitempty"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  string     `json:"created_at"`
}

// CreateStaffInput is the payload for creating a staff member.
type CreateStaffInput struct {
	BranchID *uuid.UUID `json:"branch_id"`
	Name     string     `json:"name"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Role     string     `json:"role"`
}

// UpdateStaffInput is the payload for updating a staff member.
type UpdateStaffInput struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
	BranchID *string `json:"branch_id"`
}

// Service handles staff operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new staff service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all staff for the given business.
func (s *Service) List(businessID uuid.UUID) ([]Staff, error) {
	rows, err := s.db.Query(
		`SELECT id, business_id, branch_id, name, email, role, is_active, created_at
		 FROM users WHERE business_id = $1 ORDER BY created_at`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	defer rows.Close()

	var staff []Staff
	for rows.Next() {
		var u Staff
		var branchID sql.NullString
		if err := rows.Scan(&u.ID, &u.BusinessID, &branchID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		if branchID.Valid {
			parsed, _ := uuid.Parse(branchID.String)
			u.BranchID = &parsed
		}
		staff = append(staff, u)
	}
	if staff == nil {
		staff = []Staff{}
	}
	return staff, nil
}

// Create creates a new cashier staff member.
func (s *Service) Create(businessID uuid.UUID, input CreateStaffInput) (*Staff, error) {
	if input.Name == "" || input.Email == "" || input.Password == "" {
		return nil, errors.New("name, email, and password are required")
	}
	if input.Role != "cashier" {
		return nil, errors.New("only cashier role can be created via this endpoint")
	}
	if input.BranchID == nil {
		return nil, errors.New("cashier must have a branch_id")
	}
	if len(input.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	var st Staff
	var branchID sql.NullString
	err = s.db.QueryRow(
		`INSERT INTO users (business_id, branch_id, name, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5, 'cashier')
		 RETURNING id, business_id, branch_id, name, email, role, is_active, created_at`,
		businessID, input.BranchID, input.Name, input.Email, string(hash),
	).Scan(&st.ID, &st.BusinessID, &branchID, &st.Name, &st.Email, &st.Role, &st.IsActive, &st.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create staff: %w", err)
	}
	if branchID.Valid {
		parsed, _ := uuid.Parse(branchID.String)
		st.BranchID = &parsed
	}
	return &st, nil
}

// Update updates a staff member. Cannot deactivate owner.
func (s *Service) Update(businessID, staffID uuid.UUID, input UpdateStaffInput) (*Staff, error) {
	var st Staff
	var branchID sql.NullString
	row := s.db.QueryRow(
		`SELECT id, business_id, branch_id, name, email, role, is_active, created_at
		 FROM users WHERE id = $1 AND business_id = $2`, staffID, businessID)
	if err := row.Scan(&st.ID, &st.BusinessID, &branchID, &st.Name, &st.Email, &st.Role, &st.IsActive, &st.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("staff not found")
		}
		return nil, err
	}
	if branchID.Valid {
		parsed, _ := uuid.Parse(branchID.String)
		st.BranchID = &parsed
	}

	if input.Name != nil {
		st.Name = *input.Name
	}
	if input.IsActive != nil {
		if st.Role == "owner" && !*input.IsActive {
			return nil, errors.New("cannot deactivate owner account")
		}
		st.IsActive = *input.IsActive
	}

	newBranchID := sql.NullString{}
	if st.BranchID != nil {
		newBranchID = sql.NullString{String: st.BranchID.String(), Valid: true}
	}
	if input.BranchID != nil {
		newBranchID = sql.NullString{String: *input.BranchID, Valid: true}
	}

	_, err := s.db.Exec(
		`UPDATE users SET name=$1, is_active=$2, branch_id=$3 WHERE id=$4`,
		st.Name, st.IsActive, newBranchID, st.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update staff: %w", err)
	}
	return &st, nil
}
