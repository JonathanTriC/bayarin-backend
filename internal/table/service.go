package table

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Table represents a table entity.
type Table struct {
	ID           uuid.UUID   `json:"id"`
	BranchID     uuid.UUID   `json:"branch_id"`
	Name         string      `json:"name"`
	QRCode       string      `json:"qr_code"`
	Status       string      `json:"status"` // "available", "occupied", "reserved"
	ReservedBy   *uuid.UUID  `json:"reserved_by,omitempty"`
	ReservedNote *string     `json:"reserved_note,omitempty"`
	UpdatedAt    *string     `json:"updated_at,omitempty"`
	CreatedAt    string      `json:"created_at"`
}

// CreateTableInput is the payload for creating a table.
type CreateTableInput struct {
	BranchID uuid.UUID `json:"branch_id"`
	Name     string    `json:"name"`
	QRCode   string    `json:"qr_code"`
}

type ReserveTableInput struct {
	Note string `json:"note"`
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
			`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
			 FROM tables t
			 JOIN branches b ON b.id = t.branch_id
			 WHERE b.business_id = $1 AND t.branch_id = $2
			 ORDER BY t.name`, businessID, *branchIDFilter)
	} else {
		rows, err = s.db.Query(
			`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
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
		var rBy sql.NullString
		var rNote sql.NullString
		var upAt sql.NullString
		if err := rows.Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		if rBy.Valid {
			pid, _ := uuid.Parse(rBy.String)
			t.ReservedBy = &pid
		}
		if rNote.Valid {
			t.ReservedNote = &rNote.String
		}
		if upAt.Valid {
			t.UpdatedAt = &upAt.String
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
	var rBy, rNote, upAt sql.NullString
	err := s.db.QueryRow(
		`INSERT INTO tables (branch_id, name, qr_code) VALUES ($1, $2, $3)
		 RETURNING id, branch_id, name, qr_code, status, reserved_by, reserved_note, updated_at, created_at`,
		input.BranchID, input.Name, input.QRCode,
	).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &t, nil
}

// Update applies partial updates to a table.
func (s *Service) Update(businessID, tableID uuid.UUID, input UpdateTableInput) (*Table, error) {
	var t Table
	var rBy, rNote, upAt sql.NullString
	row := s.db.QueryRow(
		`SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
		 FROM tables t
		 JOIN branches b ON b.id = t.branch_id
		 WHERE t.id = $1 AND b.business_id = $2`, tableID, businessID)
	if err := row.Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt); err != nil {
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

// Search returns tables whose name matches query (ILIKE), scoped to the business.
// Optionally filters by branch_id and/or status.
func (s *Service) Search(businessID uuid.UUID, query string, branchIDFilter *uuid.UUID, status string) ([]Table, error) {
	like := "%" + strings.TrimSpace(query) + "%"
	args := []interface{}{businessID, like}

	sqlStr := `SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
	           FROM tables t
	           JOIN branches b ON b.id = t.branch_id
	           WHERE b.business_id = $1 AND t.name ILIKE $2`

	if branchIDFilter != nil {
		args = append(args, *branchIDFilter)
		sqlStr += fmt.Sprintf(" AND t.branch_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		sqlStr += fmt.Sprintf(" AND t.status = $%d", len(args))
	}
	sqlStr += " ORDER BY t.name"

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var t Table
		var rBy sql.NullString
		var rNote sql.NullString
		var upAt sql.NullString
		if err := rows.Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		if rBy.Valid {
			pid, _ := uuid.Parse(rBy.String)
			t.ReservedBy = &pid
		}
		if rNote.Valid {
			t.ReservedNote = &rNote.String
		}
		if upAt.Valid {
			t.UpdatedAt = &upAt.String
		}
		tables = append(tables, t)
	}
	if tables == nil {
		tables = []Table{}
	}
	return tables, nil
}

// Reserve marks a table as reserved.
func (s *Service) Reserve(businessID uuid.UUID, tableID uuid.UUID, branchID uuid.UUID, reservedBy uuid.UUID, note string) (*Table, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var t Table
	var rBy, rNote, upAt sql.NullString
	err = tx.QueryRow(`
		SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
		FROM tables t
		JOIN branches b ON b.id = t.branch_id
		WHERE t.id = $1 AND b.business_id = $2 FOR UPDATE
	`, tableID, businessID).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("table not found")
		}
		return nil, fmt.Errorf("lock table: %w", err)
	}

	if t.BranchID != branchID {
		return nil, errors.New("table does not belong to this branch")
	}

	if t.Status == "occupied" {
		return nil, errors.New("table is currently occupied")
	}

	err = tx.QueryRow(`
		UPDATE tables
		SET status = 'reserved', reserved_by = $2, reserved_note = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, branch_id, name, qr_code, status, reserved_by, reserved_note, updated_at, created_at
	`, tableID, reservedBy, note).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("reserve table: %w", err)
	}

	if rBy.Valid {
		pid, _ := uuid.Parse(rBy.String)
		t.ReservedBy = &pid
	}
	if rNote.Valid {
		t.ReservedNote = &rNote.String
	}
	if upAt.Valid {
		t.UpdatedAt = &upAt.String
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &t, nil
}

// ClearStatus resets an occupied or reserved table to available.
func (s *Service) ClearStatus(businessID uuid.UUID, tableID uuid.UUID, branchID uuid.UUID) (*Table, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var t Table
	var rBy, rNote, upAt sql.NullString
	err = tx.QueryRow(`
		SELECT t.id, t.branch_id, t.name, t.qr_code, t.status, t.reserved_by, t.reserved_note, t.updated_at, t.created_at
		FROM tables t
		JOIN branches b ON b.id = t.branch_id
		WHERE t.id = $1 AND b.business_id = $2 FOR UPDATE
	`, tableID, businessID).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("table not found")
		}
		return nil, fmt.Errorf("lock table: %w", err)
	}

	if t.BranchID != branchID {
		return nil, errors.New("table does not belong to this branch")
	}

	if t.Status == "occupied" {
		var activeOrderID uuid.UUID
		err := tx.QueryRow(`SELECT id FROM orders WHERE table_id = $1 AND status = 'open' LIMIT 1`, tableID).Scan(&activeOrderID)
		if err == nil {
			return nil, errors.New("cannot clear table with an active order — complete payment first")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("query active order: %w", err)
		}
	}

	err = tx.QueryRow(`
		UPDATE tables
		SET status = 'available', reserved_by = NULL, reserved_note = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING id, branch_id, name, qr_code, status, reserved_by, reserved_note, updated_at, created_at
	`, tableID).Scan(&t.ID, &t.BranchID, &t.Name, &t.QRCode, &t.Status, &rBy, &rNote, &upAt, &t.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("clear table status: %w", err)
	}
	
	if upAt.Valid {
		t.UpdatedAt = &upAt.String
	}
	
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &t, nil
}
