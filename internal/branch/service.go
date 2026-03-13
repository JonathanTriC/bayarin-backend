package branch

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Branch represents a branch entity.
type Branch struct {
	ID         uuid.UUID `json:"id"`
	BusinessID uuid.UUID `json:"business_id"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  string    `json:"created_at"`
}

// CreateBranchInput is the payload for creating a branch.
type CreateBranchInput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// UpdateBranchInput is the payload for updating a branch.
type UpdateBranchInput struct {
	Name     *string `json:"name"`
	Address  *string `json:"address"`
	IsActive *bool   `json:"is_active"`
}

// Service handles branch operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new branch service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all branches for the given business.
func (s *Service) List(businessID uuid.UUID) ([]Branch, error) {
	rows, err := s.db.Query(
		`SELECT id, business_id, name, address, is_active, created_at
		 FROM branches WHERE business_id = $1 ORDER BY created_at`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	defer rows.Close()

	var branches []Branch
	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.ID, &b.BusinessID, &b.Name, &b.Address, &b.IsActive, &b.CreatedAt); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	if branches == nil {
		branches = []Branch{}
	}
	return branches, nil
}

// Create inserts a new branch.
func (s *Service) Create(businessID uuid.UUID, input CreateBranchInput) (*Branch, error) {
	if input.Name == "" {
		return nil, errors.New("branch name is required")
	}
	var b Branch
	err := s.db.QueryRow(
		`INSERT INTO branches (business_id, name, address) VALUES ($1, $2, $3)
		 RETURNING id, business_id, name, address, is_active, created_at`,
		businessID, input.Name, input.Address,
	).Scan(&b.ID, &b.BusinessID, &b.Name, &b.Address, &b.IsActive, &b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	return &b, nil
}

// Update applies partial updates to a branch.
func (s *Service) Update(businessID, branchID uuid.UUID, input UpdateBranchInput) (*Branch, error) {
	var b Branch
	row := s.db.QueryRow(
		`SELECT id, business_id, name, address, is_active, created_at
		 FROM branches WHERE id = $1 AND business_id = $2`, branchID, businessID)
	if err := row.Scan(&b.ID, &b.BusinessID, &b.Name, &b.Address, &b.IsActive, &b.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	if input.Name != nil {
		b.Name = *input.Name
	}
	if input.Address != nil {
		b.Address = *input.Address
	}
	if input.IsActive != nil {
		b.IsActive = *input.IsActive
	}

	_, err := s.db.Exec(
		`UPDATE branches SET name=$1, address=$2, is_active=$3 WHERE id=$4`,
		b.Name, b.Address, b.IsActive, b.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update branch: %w", err)
	}
	return &b, nil
}
