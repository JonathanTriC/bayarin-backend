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

type ModifierOptionResponse struct {
	ID         uuid.UUID `json:"id"`
	GroupID    uuid.UUID `json:"group_id"`
	Name       string    `json:"name"`
	ExtraPrice float64   `json:"extra_price"`
}

type ModifierGroupResponse struct {
	ID         uuid.UUID                `json:"id"`
	Name       string                   `json:"name"`
	IsRequired bool                     `json:"is_required"`
	MaxSelect  int                      `json:"max_select"`
	Options    []ModifierOptionResponse `json:"options"`
}

type MenuItemResponse struct {
	ID          uuid.UUID               `json:"id"`
	BusinessID  uuid.UUID               `json:"business_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Price       float64                 `json:"price"`
	Category    string                  `json:"category"`
	IsAvailable bool                    `json:"is_available"`
	Modifiers   []ModifierGroupResponse `json:"modifiers"`
	CreatedAt   string                  `json:"created_at"`
}

// CreateMenuItemInput is the payload for creating a menu item.
type CreateMenuItemInput struct {
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	Price            float64     `json:"price"`
	Category         string      `json:"category"`
	IsAvailable      bool        `json:"is_available"`
	ModifierGroupIDs []uuid.UUID `json:"modifier_group_ids"`
}

// UpdateMenuItemInput is the payload for updating a menu item.
type UpdateMenuItemInput struct {
	Name             *string     `json:"name"`
	Description      *string     `json:"description"`
	Price            *float64    `json:"price"`
	Category         *string     `json:"category"`
	IsAvailable      *bool       `json:"is_available"`
	ModifierGroupIDs []uuid.UUID `json:"modifier_group_ids"`
}

// Service handles menu item operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new menu service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) getModifiersOptions(menuItemID uuid.UUID) ([]ModifierGroupResponse, error) {
	rows, err := s.db.Query(`
		SELECT 
			mg.id AS group_id, mg.name AS group_name, mg.is_required, mg.max_select,
			mo.id AS option_id, mo.name AS option_name, mo.extra_price
		FROM menu_item_modifiers mim
		JOIN modifier_groups mg ON mg.id = mim.modifier_group_id
		JOIN modifier_options mo ON mo.group_id = mg.id
		WHERE mim.menu_item_id = $1
		ORDER BY mg.name, mo.name
	`, menuItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupMap := make(map[uuid.UUID]*ModifierGroupResponse)
	var orderedGroups []*ModifierGroupResponse

	for rows.Next() {
		var gID, oID uuid.UUID
		var gName, oName string
		var isRequired bool
		var maxSelect int
		var extraPrice float64

		if err := rows.Scan(&gID, &gName, &isRequired, &maxSelect, &oID, &oName, &extraPrice); err != nil {
			return nil, err
		}

		if _, exists := groupMap[gID]; !exists {
			group := &ModifierGroupResponse{
				ID:         gID,
				Name:       gName,
				IsRequired: isRequired,
				MaxSelect:  maxSelect,
				Options:    []ModifierOptionResponse{},
			}
			groupMap[gID] = group
			orderedGroups = append(orderedGroups, group)
		}

		groupMap[gID].Options = append(groupMap[gID].Options, ModifierOptionResponse{
			ID:         oID,
			GroupID:    gID,
			Name:       oName,
			ExtraPrice: extraPrice,
		})
	}

	result := make([]ModifierGroupResponse, 0, len(orderedGroups))
	for _, g := range orderedGroups {
		result = append(result, *g)
	}

	return result, nil
}

// List returns all menu items for the given business.
func (s *Service) List(businessID uuid.UUID) ([]MenuItemResponse, error) {
	rows, err := s.db.Query(
		`SELECT id, business_id, name, description, price, category, is_available, created_at
		 FROM menu_items WHERE business_id = $1 ORDER BY category, name`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}
	defer rows.Close()

	var items []MenuItemResponse
	for rows.Next() {
		var m MenuItemResponse
		if err := rows.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.CreatedAt); err != nil {
			return nil, err
		}

		mods, err := s.getModifiersOptions(m.ID)
		if err != nil {
			return nil, err
		}
		if mods == nil {
			mods = []ModifierGroupResponse{}
		}
		m.Modifiers = mods

		items = append(items, m)
	}
	if items == nil {
		items = []MenuItemResponse{}
	}
	return items, nil
}

// Create inserts a new menu item.
func (s *Service) Create(businessID uuid.UUID, input CreateMenuItemInput) (*MenuItemResponse, error) {
	if input.Name == "" {
		return nil, errors.New("menu item name is required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var m MenuItemResponse
	err = tx.QueryRow(
		`INSERT INTO menu_items (business_id, name, description, price, category, is_available)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, business_id, name, description, price, category, is_available, created_at`,
		businessID, input.Name, input.Description, input.Price, input.Category, input.IsAvailable,
	).Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create menu item: %w", err)
	}

	for _, groupID := range input.ModifierGroupIDs {
		// Validate that the modifier group belongs to the business
		var valid bool
		err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM modifier_groups WHERE id = $1 AND business_id = $2)`, groupID, businessID).Scan(&valid)
		if err != nil {
			return nil, fmt.Errorf("validate modifier group: %w", err)
		}
		if !valid {
			return nil, errors.New("modifier group not found or belongs to another business")
		}

		_, err = tx.Exec(`
			INSERT INTO menu_item_modifiers (menu_item_id, modifier_group_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, m.ID, groupID)
		if err != nil {
			return nil, fmt.Errorf("link modifier group: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	mods, _ := s.getModifiersOptions(m.ID)
	if mods == nil {
		mods = []ModifierGroupResponse{}
	}
	m.Modifiers = mods

	return &m, nil
}

// Update applies partial updates to a menu item.
func (s *Service) Update(businessID, itemID uuid.UUID, input UpdateMenuItemInput) (*MenuItemResponse, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var m MenuItemResponse
	row := tx.QueryRow(
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

	_, err = tx.Exec(
		`UPDATE menu_items SET name=$1, description=$2, price=$3, category=$4, is_available=$5 WHERE id=$6`,
		m.Name, m.Description, m.Price, m.Category, m.IsAvailable, m.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update menu item: %w", err)
	}

	if input.ModifierGroupIDs != nil {
		_, err = tx.Exec(`DELETE FROM menu_item_modifiers WHERE menu_item_id = $1`, m.ID)
		if err != nil {
			return nil, fmt.Errorf("delete old modifiers: %w", err)
		}

		for _, groupID := range input.ModifierGroupIDs {
			var valid bool
			err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM modifier_groups WHERE id = $1 AND business_id = $2)`, groupID, businessID).Scan(&valid)
			if err != nil {
				return nil, fmt.Errorf("validate modifier group: %w", err)
			}
			if !valid {
				return nil, errors.New("modifier group not found or belongs to another business")
			}

			_, err = tx.Exec(`
				INSERT INTO menu_item_modifiers (menu_item_id, modifier_group_id)
				VALUES ($1, $2) ON CONFLICT DO NOTHING
			`, m.ID, groupID)
			if err != nil {
				return nil, fmt.Errorf("link modifier group: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	mods, _ := s.getModifiersOptions(m.ID)
	if mods == nil {
		mods = []ModifierGroupResponse{}
	}
	m.Modifiers = mods

	return &m, nil
}
