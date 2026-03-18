package menu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bayarin/backend/config"
	"github.com/bayarin/backend/internal/db/sqlcgen"
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
	ImageURL    *string   `json:"image_url,omitempty"`
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
	ImageURL    *string                 `json:"image_url,omitempty"`
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
	db  *sql.DB
	cfg *config.Config
}

// NewService creates a new menu service.
func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

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
		`SELECT id, business_id, name, description, price, category, is_available, image_url, created_at
		 FROM menu_items WHERE business_id = $1 ORDER BY category, name`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}
	defer rows.Close()

	var items []MenuItemResponse
	for rows.Next() {
		var m MenuItemResponse
		if err := rows.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.ImageURL, &m.CreatedAt); err != nil {
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
		 RETURNING id, business_id, name, description, price, category, is_available, image_url, created_at`,
		businessID, input.Name, input.Description, input.Price, input.Category, input.IsAvailable,
	).Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.ImageURL, &m.CreatedAt)
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
		`SELECT id, business_id, name, description, price, category, is_available, image_url, created_at
		 FROM menu_items WHERE id = $1 AND business_id = $2`, itemID, businessID)
	if err := row.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.ImageURL, &m.CreatedAt); err != nil {
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

// Search returns menu items whose name or description match query (ILIKE).
// Optionally filters by category. Results are scoped to the given business.
func (s *Service) Search(businessID uuid.UUID, query, category string) ([]MenuItemResponse, error) {
	like := "%" + strings.TrimSpace(query) + "%"
	args := []interface{}{businessID, like}

	sqlStr := `SELECT id, business_id, name, description, price, category, is_available, image_url, created_at
	           FROM menu_items
	           WHERE business_id = $1 AND (name ILIKE $2 OR description ILIKE $2)`

	if category != "" {
		args = append(args, category)
		sqlStr += fmt.Sprintf(" AND category = $%d", len(args))
	}
	sqlStr += " ORDER BY category, name"

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search menu items: %w", err)
	}
	defer rows.Close()

	var items []MenuItemResponse
	for rows.Next() {
		var m MenuItemResponse
		if err := rows.Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.ImageURL, &m.CreatedAt); err != nil {
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

// Categories returns the distinct list of categories in use for the given business.
// The list is always live — a new menu item with a new category appears here automatically.
func (s *Service) Categories(businessID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT category FROM menu_items
		 WHERE business_id = $1 AND category <> ''
		 ORDER BY category`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []string{}
	}
	return categories, nil
}

// UploadMenuItemImage handles image validation, upload, and updating the database.
func (s *Service) UploadMenuItemImage(ctx context.Context, itemID, businessID uuid.UUID, fileData []byte, filename string, fileSize int64) (*MenuItemResponse, error) {
	// 1. Validate file extension
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil, errors.New("only JPG and PNG images are allowed")
	}

	// 2. Validate file size (max 5MB)
	if fileSize > 5*1024*1024 {
		return nil, errors.New("image file must not exceed 5MB")
	}

	// 3. Check menu item exists
	var m MenuItemResponse
	err := s.db.QueryRow(`
		SELECT id, business_id, name, description, price, category, is_available, image_url, created_at
		FROM menu_items WHERE id = $1 AND business_id = $2
	`, itemID, businessID).Scan(&m.ID, &m.BusinessID, &m.Name, &m.Description, &m.Price, &m.Category, &m.IsAvailable, &m.ImageURL, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("menu item not found")
		}
		return nil, fmt.Errorf("query menu item: %w", err)
	}

	// 4. Delete old image if present
	if m.ImageURL != nil && *m.ImageURL != "" {
		// Public url format: {SUPABASE_URL}/storage/v1/object/public/{bucket}/{path}
		parts := strings.Split(*m.ImageURL, "/public/"+s.cfg.SupabaseMenuBucket+"/")
		if len(parts) == 2 {
			oldPath := parts[1]
			_ = DeleteMenuImage(s.cfg, oldPath) // best effort
		}
	}

	// 5. Generate storage path
	newID := uuid.New()
	storagePath := fmt.Sprintf("menu/%s/%s/%s%s", businessID.String(), itemID.String(), newID.String(), ext)

	// 6. Determine content type
	contentType := "image/jpeg"
	if ext == ".png" {
		contentType = "image/png"
	}

	// 7. Upload to Supabase
	publicURL, err := UploadMenuImage(s.cfg, storagePath, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// 8. Update database using sqlc generated query (here we use raw SQL to keep transactions tight or call sqlc)
	queries := sqlcgen.New(s.db)
	_, err = queries.UpdateMenuItemImage(ctx, sqlcgen.UpdateMenuItemImageParams{
		ID:         itemID,
		BusinessID: businessID,
		ImageUrl:   sql.NullString{String: publicURL, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("update menu item image: %w", err)
	}

	// 9. Fetch and return full MenuItemResponse
	m.ImageURL = &publicURL

	mods, _ := s.getModifiersOptions(m.ID)
	if mods == nil {
		mods = []ModifierGroupResponse{}
	}
	m.Modifiers = mods

	return &m, nil
}
