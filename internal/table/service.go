package table

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Table represents a table entity.
type Table struct {
	ID        uuid.UUID `json:"id"`
	BranchID  uuid.UUID `json:"branch_id"`
	Name      string    `json:"name"`
	QRCode    string    `json:"qr_code"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

// CreateTableInput is the payload for creating a table.
type CreateTableInput struct {
	BranchID uuid.UUID `json:"branch_id"`
	Name     string    `json:"name"`
	QRCode   string    `json:"qr_code"`
}

// UpdateTableInput is the payload for updating a table.
type UpdateTableInput struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
	QRCode *string `json:"qr_code"`
}

// Service handles table operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new table service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all tables for a given branch (filtered by branch_id query param if provided),
// otherwise returns all tables across all branches in the business.
func (s *Service) List(businessID uuid.UUID, branchIDFilter *uuid.UUID) ([]Table, error) {
	var rows *sql.Rows
	var err error

	if branchIDFilter != nil {
		rows, err = s.db.Query(
			`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.created_at
			 FROM tables t
			 JOIN branches b ON b.id = t.branch_id
			 WHERE b.business_id = $1 AND t.branch_id = $2
			 ORDER BY t.name`, businessID, *branchIDFilter)
	} else {
		rows, err = s.db.Query(
			`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.created_at
			 FROM tables t
			 JOIN branches b ON b.id = t.branch_id
			 WHERE b.business_id = $1
			 ORDER BY t.name`, businessID)
	}
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	if tables == nil {
		tables = []Table{}
	}
	return tables, nil
}

// Create inserts a new table, validating that the branch belongs to the business.
func (s *Service) Create(businessID uuid.UUID, input CreateTableInput) (*Table, error) {
	if input.Name == "" {
		return nil, errors.New("table name is required")
	}

	// Ensure branch belongs to business.
	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM branches WHERE id = $1 AND business_id = $2`,
		input.BranchID, businessID,
	).Scan(&count); err != nil || count == 0 {
		return nil, errors.New("branch not found or does not belong to this business")
	}

	var t Table
	err := s.db.QueryRow(
		`INSERT INTO tables (branch_id, name, qr_code) VALUES ($1, $2, $3)
		 RETURNING id, branch_id, name, qr_code, status, created_at`,
		input.BranchID, input.Name, input.QRCode,
	).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &t, nil
}

// Update applies partial updates to a table.
func (s *Service) Update(businessID, tableID uuid.UUID, input UpdateTableInput) (*Table, error) {
	var t Table
	row := s.db.QueryRow(
		`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.created_at
		 FROM tables t
		 JOIN branches b ON b.id = t.branch_id
		 WHERE t.id = $1 AND b.business_id = $2`, tableID, businessID)
	if err := row.Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("table not found")
		}
		return nil, err
	}

	if input.Name != nil {
		t.Name = *input.Name
	}
	if input.Status != nil {
		if *input.Status != "available" && *input.Status != "occupied" {
			return nil, errors.New("status must be 'available' or 'occupied'")
		}
		t.Status = *input.Status
	}
	if input.QRCode != nil {
		t.QRCode = *input.QRCode
	}

	_, err := s.db.Exec(
		`UPDATE tables SET name=$1, status=$2, qr_code=$3 WHERE id=$4`,
		t.Name, t.Status, t.QRCode, t.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update table: %w", err)
	}
	return &t, nil
}
