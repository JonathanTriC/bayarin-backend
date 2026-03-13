package menu

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// MenuItem represents a menu item entity.
type MenuItem struct {
	ID          uuid.UUID `json:"id"`
	BusinessID  uuid.UUID `json:"business_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   string    `json:"created_at"`
}

// CreateMenuItemInput is the payload for creating a menu item.
type CreateMenuItemInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

// UpdateMenuItemInput is the payload for updating a menu item.
type UpdateMenuItemInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Category    *string  `json:"category"`
	IsAvailable *bool    `json:"is_available"`
}

// Service handles menu item operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new menu service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all menu items for the given business.
func (s *Service) List(businessID uuid.UUID) ([]MenuItem, error) {
	rows, err := s.db.Query(
		`SELECT id, business_id, name, description, price, category, is_available, created_at
		 FROM menu_items WHERE business_id = $1 ORDER BY category, name`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}
	defer rows.Close()

	var items []MenuItem
	for rows.Next() {
		var m MenuItem
		if err := rows.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	if items == nil {
		items = []MenuItem{}
	}
	return items, nil
}

// Create inserts a new menu item.
func (s *Service) Create(businessID uuid.UUID, input CreateMenuItemInput) (*MenuItem, error) {
	if input.Name == "" {
		return nil, errors.New("menu item name is required")
	}
	var m MenuItem
	err := s.db.QueryRow(
		`INSERT INTO menu_items (business_id, name, description, price, category)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, business_id, name, description, price, category, is_available, created_at`,
		businessID, input.Name, input.Description, input.Price, input.Category,
	).Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create menu item: %w", err)
	}
	return &m, nil
}

// Update applies partial updates to a menu item.
func (s *Service) Update(businessID, itemID uuid.UUID, input UpdateMenuItemInput) (*MenuItem, error) {
	var m MenuItem
	row := s.db.QueryRow(
		`SELECT id, business_id, name, description, price, category, is_available, created_at
		 FROM menu_items WHERE id = $1 AND business_id = $2`, itemID, businessID)
	if err := row.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("menu item not found")
		}
		return nil, err
	}

	if input.Name != nil {
		m.Name = *input.Name
	}
	if input.Description != nil {
		m.Description = *input.Description
	}
	if input.Price != nil {
		m.Price = *input.Price
	}
	if input.Category != nil {
		m.Category = *input.Category
	}
	if input.IsAvailable != nil {
		m.IsAvailable = *input.IsAvailable
	}

	_, err := s.db.Exec(
		`UPDATE menu_items SET name=$1, description=$2, price=$3, category=$4, is_available=$5 WHERE id=$6`,
		m.Name, m.Description, m.Price, m.Category, m.IsAvailable, m.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update menu item: %w", err)
	}
	return &m, nil
}
