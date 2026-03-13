package modifier

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ModifierGroup represents a modifier group entity.
type ModifierGroup struct {
	ID         uuid.UUID        `json:"id"`
	BusinessID uuid.UUID        `json:"business_id"`
	Name       string           `json:"name"`
	IsRequired bool             `json:"is_required"`
	MaxSelect  int              `json:"max_select"`
	Options    []ModifierOption `json:"options"`
}

// ModifierOption represents an option within a group.
type ModifierOption struct {
	ID         uuid.UUID `json:"id"`
	GroupID    uuid.UUID `json:"group_id"`
	Name       string    `json:"name"`
	ExtraPrice float64   `json:"extra_price"`
}

// CreateModifierGroupInput is the payload for creating a modifier group with options.
type CreateModifierGroupInput struct {
	Name       string `json:"name"`
	IsRequired bool   `json:"is_required"`
	MaxSelect  int    `json:"max_select"`
	Options    []struct {
		Name       string  `json:"name"`
		ExtraPrice float64 `json:"extra_price"`
	} `json:"options"`
}

// UpdateModifierGroupInput is the payload for updating a modifier group.
type UpdateModifierGroupInput struct {
	Name       *string `json:"name"`
	IsRequired *bool   `json:"is_required"`
	MaxSelect  *int    `json:"max_select"`
}

// Service handles modifier operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new modifier service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all modifier groups (with options) for the given business.
func (s *Service) List(businessID uuid.UUID) ([]ModifierGroup, error) {
	rows, err := s.db.Query(
		`SELECT id, business_id, name, is_required, max_select
		 FROM modifier_groups WHERE business_id = $1 ORDER BY name`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list modifier groups: %w", err)
	}
	defer rows.Close()

	var groups []ModifierGroup
	for rows.Next() {
		var g ModifierGroup
		if err := rows.Scan(&g.ID, &g.BusinessID, &g.Name, &g.IsRequired, &g.MaxSelect); err != nil {
			return nil, err
		}
		// Load options for this group.
		opts, err := s.listOptions(g.ID)
		if err != nil {
			return nil, err
		}
		g.Options = opts
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []ModifierGroup{}
	}
	return groups, nil
}

func (s *Service) listOptions(groupID uuid.UUID) ([]ModifierOption, error) {
	rows, err := s.db.Query(
		`SELECT id, group_id, name, extra_price FROM modifier_options WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []ModifierOption
	for rows.Next() {
		var o ModifierOption
		if err := rows.Scan(&o.ID, &o.GroupID, &o.Name, &o.ExtraPrice); err != nil {
			return nil, err
		}
		opts = append(opts, o)
	}
	if opts == nil {
		opts = []ModifierOption{}
	}
	return opts, nil
}

// Create inserts a new modifier group with its options (in a single transaction).
func (s *Service) Create(businessID uuid.UUID, input CreateModifierGroupInput) (*ModifierGroup, error) {
	if input.Name == "" {
		return nil, errors.New("modifier group name is required")
	}
	if input.MaxSelect < 1 {
		input.MaxSelect = 1
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var g ModifierGroup
	err = tx.QueryRow(
		`INSERT INTO modifier_groups (business_id, name, is_required, max_select)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, business_id, name, is_required, max_select`,
		businessID, input.Name, input.IsRequired, input.MaxSelect,
	).Scan(&g.ID, &g.BusinessID, &g.Name, &g.IsRequired, &g.MaxSelect)
	if err != nil {
		return nil, fmt.Errorf("insert modifier group: %w", err)
	}

	for _, opt := range input.Options {
		var o ModifierOption
		err = tx.QueryRow(
			`INSERT INTO modifier_options (group_id, name, extra_price) VALUES ($1, $2, $3)
			 RETURNING id, group_id, name, extra_price`,
			g.ID, opt.Name, opt.ExtraPrice,
		).Scan(&o.ID, &o.GroupID, &o.Name, &o.ExtraPrice)
		if err != nil {
			return nil, fmt.Errorf("insert modifier option: %w", err)
		}
		g.Options = append(g.Options, o)
	}
	if g.Options == nil {
		g.Options = []ModifierOption{}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &g, nil
}

// Update applies partial updates to a modifier group.
func (s *Service) Update(businessID, groupID uuid.UUID, input UpdateModifierGroupInput) (*ModifierGroup, error) {
	var g ModifierGroup
	row := s.db.QueryRow(
		`SELECT id, business_id, name, is_required, max_select
		 FROM modifier_groups WHERE id = $1 AND business_id = $2`, groupID, businessID)
	if err := row.Scan(&g.ID, &g.BusinessID, &g.Name, &g.IsRequired, &g.MaxSelect); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("modifier group not found")
		}
		return nil, err
	}

	if input.Name != nil {
		g.Name = *input.Name
	}
	if input.IsRequired != nil {
		g.IsRequired = *input.IsRequired
	}
	if input.MaxSelect != nil {
		g.MaxSelect = *input.MaxSelect
	}

	_, err := s.db.Exec(
		`UPDATE modifier_groups SET name=$1, is_required=$2, max_select=$3 WHERE id=$4`,
		g.Name, g.IsRequired, g.MaxSelect, g.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update modifier group: %w", err)
	}

	opts, err := s.listOptions(g.ID)
	if err != nil {
		return nil, err
	}
	g.Options = opts
	return &g, nil
}
