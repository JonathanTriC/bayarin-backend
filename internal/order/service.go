package order

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/bayarin/backend/internal/middleware"
	"github.com/google/uuid"
)

// Order represents an order entity.
type Order struct {
	ID                   uuid.UUID   `json:"id"`
	BusinessID           uuid.UUID   `json:"business_id"`
	BranchID             uuid.UUID   `json:"branch_id"`
	CashierID            uuid.UUID   `json:"cashier_id"`
	TableID              *uuid.UUID  `json:"table_id,omitempty"`
	Type                 string      `json:"type"`
	CustomerName         string      `json:"customer_name"`
	Status               string      `json:"status"`
	Subtotal             float64     `json:"subtotal"`
	TaxAmount            float64     `json:"tax_amount"`
	ServiceChargeAmount  float64     `json:"service_charge_amount"`
	Total                float64     `json:"total"`
	CreatedAt            string      `json:"created_at"`
	Items                []OrderItem `json:"items,omitempty"`
}

// OrderItem represents a single line item in an order.
type OrderItem struct {
	ID           uuid.UUID           `json:"id"`
	OrderID      uuid.UUID           `json:"order_id"`
	MenuItemID   uuid.UUID           `json:"menu_item_id"`
	Quantity     int                 `json:"quantity"`
	UnitPrice    float64             `json:"unit_price"`
	Notes        string              `json:"notes"`
	Subtotal     float64             `json:"subtotal"`
	Modifiers    []OrderItemModifier `json:"modifiers,omitempty"`
}

// OrderItemModifier represents a modifier applied to an order item.
type OrderItemModifier struct {
	ID               uuid.UUID `json:"id"`
	OrderItemID      uuid.UUID `json:"order_item_id"`
	ModifierOptionID uuid.UUID `json:"modifier_option_id"`
	ExtraPrice       float64   `json:"extra_price"`
}

// CreateOrderInput is the payload for creating a new order.
type CreateOrderInput struct {
	BranchID     uuid.UUID  `json:"branch_id"`
	TableID      *uuid.UUID `json:"table_id"`
	Type         string     `json:"type"`
	CustomerName string     `json:"customer_name"`
}

// UpdateOrderInput is the payload for updating an order status.
type UpdateOrderInput struct {
	Status       *string `json:"status"`
	CustomerName *string `json:"customer_name"`
}

// AddOrderItemInput is the payload for adding an item to an order.
type AddOrderItemInput struct {
	MenuItemID        uuid.UUID   `json:"menu_item_id"`
	Quantity          int         `json:"quantity"`
	Notes             string      `json:"notes"`
	ModifierOptionIDs []uuid.UUID `json:"modifier_option_ids"`
}

// UpdateOrderItemInput is the payload for updating an existing order item.
type UpdateOrderItemInput struct {
	Quantity          *int        `json:"quantity"`
	Notes             *string     `json:"notes"`
	ModifierOptionIDs []uuid.UUID `json:"modifier_option_ids"`
}

// Service handles order operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new order service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns orders for the business, optionally filtered by status.
func (s *Service) List(auth middleware.AuthContext, statusFilter string) ([]Order, error) {
	query := `SELECT id, business_id, branch_id, cashier_id, table_id, type, customer_name,
	                 status, subtotal, tax_amount, service_charge_amount, total, created_at
	          FROM orders WHERE business_id = $1`
	args := []interface{}{auth.BusinessID}

	if auth.Role == "cashier" && auth.BranchID != nil {
		query += " AND branch_id = $2"
		args = append(args, *auth.BranchID)
		if statusFilter != "" {
			query += " AND status = $3"
			args = append(args, statusFilter)
		}
	} else {
		if statusFilter != "" {
			query += " AND status = $2"
			args = append(args, statusFilter)
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var tableID sql.NullString
		if err := rows.Scan(&o.ID, &o.BusinessID, &o.BranchID, &o.CashierID, &tableID,
			&o.Type, &o.CustomerName, &o.Status, &o.Subtotal, &o.TaxAmount,
			&o.ServiceChargeAmount, &o.Total, &o.CreatedAt); err != nil {
			return nil, err
		}
		if tableID.Valid {
			parsed, _ := uuid.Parse(tableID.String)
			o.TableID = &parsed
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []Order{}
	}
	return orders, nil
}

// Create inserts a new order.
func (s *Service) Create(auth middleware.AuthContext, input CreateOrderInput) (*Order, error) {
	if input.Type != "dine_in" && input.Type != "takeaway" {
		return nil, errors.New("type must be 'dine_in' or 'takeaway'")
	}

	var o Order
	var tableID sql.NullString
	err := s.db.QueryRow(
		`INSERT INTO orders (business_id, branch_id, cashier_id, table_id, type, customer_name)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, business_id, branch_id, cashier_id, table_id, type, customer_name,
		           status, subtotal, tax_amount, service_charge_amount, total, created_at`,
		auth.BusinessID, input.BranchID, auth.UserID, input.TableID, input.Type, input.CustomerName,
	).Scan(&o.ID, &o.BusinessID, &o.BranchID, &o.CashierID, &tableID,
		&o.Type, &o.CustomerName, &o.Status, &o.Subtotal, &o.TaxAmount,
		&o.ServiceChargeAmount, &o.Total, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	if tableID.Valid {
		parsed, _ := uuid.Parse(tableID.String)
		o.TableID = &parsed
	}

	// Mark table as occupied if dine_in.
	if input.Type == "dine_in" && input.TableID != nil {
		_, _ = s.db.Exec(`UPDATE tables SET status='occupied' WHERE id=$1`, input.TableID)
	}
	return &o, nil
}

// GetByID returns a full order with its items.
func (s *Service) GetByID(businessID, orderID uuid.UUID) (*Order, error) {
	var o Order
	var tableID sql.NullString
	row := s.db.QueryRow(
		`SELECT id, business_id, branch_id, cashier_id, table_id, type, customer_name,
		        status, subtotal, tax_amount, service_charge_amount, total, created_at
		 FROM orders WHERE id = $1 AND business_id = $2`, orderID, businessID)
	if err := row.Scan(&o.ID, &o.BusinessID, &o.BranchID, &o.CashierID, &tableID,
		&o.Type, &o.CustomerName, &o.Status, &o.Subtotal, &o.TaxAmount,
		&o.ServiceChargeAmount, &o.Total, &o.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	if tableID.Valid {
		parsed, _ := uuid.Parse(tableID.String)
		o.TableID = &parsed
	}

	items, err := s.getItems(orderID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (s *Service) getItems(orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := s.db.Query(
		`SELECT id, order_id, menu_item_id, quantity, unit_price, notes, subtotal
		 FROM order_items WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Quantity,
			&item.UnitPrice, &item.Notes, &item.Subtotal); err != nil {
			return nil, err
		}
		mods, _ := s.getItemModifiers(item.ID)
		item.Modifiers = mods
		items = append(items, item)
	}
	if items == nil {
		items = []OrderItem{}
	}
	return items, nil
}

func (s *Service) getItemModifiers(itemID uuid.UUID) ([]OrderItemModifier, error) {
	rows, err := s.db.Query(
		`SELECT id, order_item_id, modifier_option_id, extra_price
		 FROM order_item_modifiers WHERE order_item_id = $1`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mods []OrderItemModifier
	for rows.Next() {
		var m OrderItemModifier
		if err := rows.Scan(&m.ID, &m.OrderItemID, &m.ModifierOptionID, &m.ExtraPrice); err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	if mods == nil {
		mods = []OrderItemModifier{}
	}
	return mods, nil
}

// Update applies partial updates to an order.
func (s *Service) Update(businessID, orderID uuid.UUID, input UpdateOrderInput) (*Order, error) {
	o, err := s.GetByID(businessID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status == "paid" || o.Status == "cancelled" {
		return nil, errors.New("cannot update a paid or cancelled order")
	}
	if input.Status != nil {
		o.Status = *input.Status
	}
	if input.CustomerName != nil {
		o.CustomerName = *input.CustomerName
	}
	_, err = s.db.Exec(
		`UPDATE orders SET status=$1, customer_name=$2 WHERE id=$3`, o.Status, o.CustomerName, o.ID)
	if err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}
	return o, nil
}

// AddItem adds a menu item (with optional modifiers) to an order and recalculates totals.
func (s *Service) AddItem(businessID, orderID uuid.UUID, input AddOrderItemInput) (*OrderItem, error) {
	o, err := s.GetByID(businessID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != "open" {
		return nil, errors.New("can only add items to an open order")
	}
	if input.Quantity < 1 {
		return nil, errors.New("quantity must be at least 1")
	}

	// Fetch menu item price.
	var unitPrice float64
	if err := s.db.QueryRow(
		`SELECT price FROM menu_items WHERE id = $1 AND business_id = $2 AND is_available = true`,
		input.MenuItemID, businessID,
	).Scan(&unitPrice); err != nil {
		return nil, errors.New("menu item not found or unavailable")
	}

	// Calculate modifier extra prices.
	modifierTotal := 0.0
	for _, modID := range input.ModifierOptionIDs {
		var ep float64
		_ = s.db.QueryRow(`SELECT extra_price FROM modifier_options WHERE id = $1`, modID).Scan(&ep)
		modifierTotal += ep
	}

	effectiveUnitPrice := unitPrice + modifierTotal
	subtotal := effectiveUnitPrice * float64(input.Quantity)

	var item OrderItem
	err = s.db.QueryRow(
		`INSERT INTO order_items (order_id, menu_item_id, quantity, unit_price, notes, subtotal)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, order_id, menu_item_id, quantity, unit_price, notes, subtotal`,
		orderID, input.MenuItemID, input.Quantity, effectiveUnitPrice, input.Notes, subtotal,
	).Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Quantity, &item.UnitPrice, &item.Notes, &item.Subtotal)
	if err != nil {
		return nil, fmt.Errorf("insert order item: %w", err)
	}

	// Insert modifiers.
	for _, modID := range input.ModifierOptionIDs {
		var ep float64
		_ = s.db.QueryRow(`SELECT extra_price FROM modifier_options WHERE id = $1`, modID).Scan(&ep)
		var mod OrderItemModifier
		_ = s.db.QueryRow(
			`INSERT INTO order_item_modifiers (order_item_id, modifier_option_id, extra_price)
			 VALUES ($1, $2, $3) RETURNING id, order_item_id, modifier_option_id, extra_price`,
			item.ID, modID, ep,
		).Scan(&mod.ID, &mod.OrderItemID, &mod.ModifierOptionID, &mod.ExtraPrice)
		item.Modifiers = append(item.Modifiers, mod)
	}

	if err := s.recalculateOrder(orderID, businessID); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem updates an existing order item.
func (s *Service) UpdateItem(businessID, orderID, itemID uuid.UUID, input UpdateOrderItemInput) (*OrderItem, error) {
	o, err := s.GetByID(businessID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != "open" {
		return nil, errors.New("can only update items on an open order")
	}

	var item OrderItem
	row := s.db.QueryRow(
		`SELECT id, order_id, menu_item_id, quantity, unit_price, notes, subtotal
		 FROM order_items WHERE id = $1 AND order_id = $2`, itemID, orderID)
	if err := row.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Quantity,
		&item.UnitPrice, &item.Notes, &item.Subtotal); err != nil {
		return nil, errors.New("order item not found")
	}

	if input.Quantity != nil {
		item.Quantity = *input.Quantity
	}
	if input.Notes != nil {
		item.Notes = *input.Notes
	}
	item.Subtotal = item.UnitPrice * float64(item.Quantity)

	_, err = s.db.Exec(
		`UPDATE order_items SET quantity=$1, notes=$2, subtotal=$3 WHERE id=$4`,
		item.Quantity, item.Notes, item.Subtotal, item.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update order item: %w", err)
	}

	if err := s.recalculateOrder(orderID, businessID); err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteItem removes an order item.
func (s *Service) DeleteItem(businessID, orderID, itemID uuid.UUID) error {
	o, err := s.GetByID(businessID, orderID)
	if err != nil {
		return err
	}
	if o.Status != "open" {
		return errors.New("can only delete items from an open order")
	}
	_, err = s.db.Exec(`DELETE FROM order_items WHERE id=$1 AND order_id=$2`, itemID, orderID)
	if err != nil {
		return fmt.Errorf("delete order item: %w", err)
	}
	return s.recalculateOrder(orderID, businessID)
}

// recalculateOrder recalculates subtotal, tax, service charge, and total for an order.
func (s *Service) recalculateOrder(orderID, businessID uuid.UUID) error {
	var subtotal float64
	if err := s.db.QueryRow(
		`SELECT COALESCE(SUM(subtotal), 0) FROM order_items WHERE order_id = $1`, orderID,
	).Scan(&subtotal); err != nil {
		return fmt.Errorf("sum order items: %w", err)
	}

	var taxPercent, svcPercent float64
	if err := s.db.QueryRow(
		`SELECT b.tax_percent, b.service_charge_percent
		 FROM orders o JOIN businesses b ON b.id = o.business_id
		 WHERE o.id = $1`, orderID,
	).Scan(&taxPercent, &svcPercent); err != nil {
		return fmt.Errorf("fetch business rates: %w", err)
	}

	taxAmount := subtotal * taxPercent / 100
	svcAmount := subtotal * svcPercent / 100
	total := subtotal + taxAmount + svcAmount

	_, err := s.db.Exec(
		`UPDATE orders SET subtotal=$1, tax_amount=$2, service_charge_amount=$3, total=$4 WHERE id=$5`,
		subtotal, taxAmount, svcAmount, total, orderID,
	)
	return err
}
