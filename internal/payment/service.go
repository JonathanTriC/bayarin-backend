package payment

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Payment represents a payment entity.
type Payment struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"order_id"`
	Method       string    `json:"method"`
	AmountPaid   float64   `json:"amount_paid"`
	ChangeAmount float64   `json:"change_amount"`
	PaidAt       string    `json:"paid_at"`
}

// PayOrderInput is the payload for paying an order.
type PayOrderInput struct {
	Method     string  `json:"method"`
	AmountPaid float64 `json:"amount_paid"`
}

// Service handles payment operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new payment service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Pay processes payment for an order in a single atomic transaction:
// 1. Lock order row (SELECT FOR UPDATE)
// 2. Recalculate: subtotal → tax → service_charge → total
// 3. Insert payment row
// 4. Insert transaction row
// 5. Update order status to 'paid'
// 6. Update table status to 'available' (if dine_in)
func (s *Service) Pay(businessID, orderID uuid.UUID, input PayOrderInput) (*Payment, error) {
	if input.Method != "cash" && input.Method != "qris" && input.Method != "transfer" {
		return nil, errors.New("method must be 'cash', 'qris', or 'transfer'")
	}
	if input.AmountPaid <= 0 {
		return nil, errors.New("amount_paid must be positive")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Step 1: Lock order row.
	var (
		currentStatus string
		orderType     string
		branchID      uuid.UUID
		tableID       sql.NullString
	)
	err = tx.QueryRow(
		`SELECT status, type, branch_id, table_id FROM orders
		 WHERE id = $1 AND business_id = $2 FOR UPDATE`, orderID, businessID,
	).Scan(&currentStatus, &orderType, &branchID, &tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, fmt.Errorf("lock order: %w", err)
	}
	if currentStatus != "open" {
		return nil, fmt.Errorf("order is %s, cannot be paid", currentStatus)
	}

	// Step 2: Recalculate totals.
	var subtotal float64
	if err := tx.QueryRow(
		`SELECT COALESCE(SUM(subtotal), 0) FROM order_items WHERE order_id = $1`, orderID,
	).Scan(&subtotal); err != nil {
		return nil, fmt.Errorf("sum items: %w", err)
	}

	var taxPercent, svcPercent float64
	if err := tx.QueryRow(
		`SELECT b.tax_percent, b.service_charge_percent
		 FROM orders o JOIN businesses b ON b.id = o.business_id
		 WHERE o.id = $1`, orderID,
	).Scan(&taxPercent, &svcPercent); err != nil {
		return nil, fmt.Errorf("fetch rates: %w", err)
	}

	taxAmount := subtotal * taxPercent / 100
	svcAmount := subtotal * svcPercent / 100
	total := subtotal + taxAmount + svcAmount

	if input.AmountPaid < total {
		return nil, fmt.Errorf("amount_paid (%.2f) is less than order total (%.2f)", input.AmountPaid, total)
	}
	changeAmount := input.AmountPaid - total

	// Step 3: Insert payment.
	var p Payment
	err = tx.QueryRow(
		`INSERT INTO payments (order_id, method, amount_paid, change_amount)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, order_id, method, amount_paid, change_amount, paid_at`,
		orderID, input.Method, input.AmountPaid, changeAmount,
	).Scan(&p.ID, &p.OrderID, &p.Method, &p.AmountPaid, &p.ChangeAmount, &p.PaidAt)
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}

	// Step 4: Insert transaction record.
	_, err = tx.Exec(
		`INSERT INTO transactions (business_id, order_id, branch_id, total)
		 SELECT business_id, id, branch_id, $1 FROM orders WHERE id = $2`,
		total, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	// Step 5: Update order to paid + store recalculated totals.
	_, err = tx.Exec(
		`UPDATE orders SET status='paid', subtotal=$1, tax_amount=$2, service_charge_amount=$3, total=$4
		 WHERE id=$5`,
		subtotal, taxAmount, svcAmount, total, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	// Step 6: Release table if dine_in.
	if orderType == "dine_in" && tableID.Valid {
		_, err = tx.Exec(`UPDATE tables SET status='available' WHERE id=$1`, tableID.String)
		if err != nil {
			return nil, fmt.Errorf("release table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &p, nil
}
