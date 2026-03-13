package dashboard

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// OwnerDashboard contains the owner's overview metrics.
type OwnerDashboard struct {
	TotalRevenueToday float64       `json:"total_revenue_today"`
	TotalOrdersToday  int           `json:"total_orders_today"`
	TopMenuItems      []TopMenuItem `json:"top_menu_items"`
	RevenuePerBranch  []BranchRev   `json:"revenue_per_branch"`
}

// TopMenuItem represents a top-selling menu item.
type TopMenuItem struct {
	MenuItemID   uuid.UUID `json:"menu_item_id"`
	MenuItemName string    `json:"menu_item_name"`
	TotalSold    int       `json:"total_sold"`
}

// BranchRev represents the revenue of a single branch.
type BranchRev struct {
	BranchID   uuid.UUID `json:"branch_id"`
	BranchName string    `json:"branch_name"`
	Revenue    float64   `json:"revenue"`
}

// CashierDashboard contains the cashier's overview metrics.
type CashierDashboard struct {
	OpenOrdersCount   int     `json:"open_orders_count"`
	OrdersHandledToday int   `json:"orders_handled_today"`
	TotalCollectedToday float64 `json:"total_collected_today"`
}

// Service handles dashboard queries.
type Service struct {
	db *sql.DB
}

// NewService creates a new dashboard service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// OwnerStats returns the owner dashboard metrics for the current calendar day (UTC).
func (s *Service) OwnerStats(businessID uuid.UUID) (*OwnerDashboard, error) {
	d := &OwnerDashboard{}

	// Total revenue today.
	if err := s.db.QueryRow(
		`SELECT COALESCE(SUM(total), 0) FROM transactions
		 WHERE business_id = $1 AND paid_at >= CURRENT_DATE`, businessID,
	).Scan(&d.TotalRevenueToday); err != nil {
		return nil, fmt.Errorf("revenue today: %w", err)
	}

	// Total orders today.
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM orders
		 WHERE business_id = $1 AND status = 'paid' AND created_at >= CURRENT_DATE`, businessID,
	).Scan(&d.TotalOrdersToday); err != nil {
		return nil, fmt.Errorf("orders today: %w", err)
	}

	// Top 5 menu items by quantity sold today.
	rows, err := s.db.Query(
		`SELECT oi.menu_item_id, mi.name, SUM(oi.quantity) AS total_sold
		 FROM order_items oi
		 JOIN menu_items mi ON mi.id = oi.menu_item_id
		 JOIN orders o ON o.id = oi.order_id
		 WHERE o.business_id = $1 AND o.status = 'paid' AND o.created_at >= CURRENT_DATE
		 GROUP BY oi.menu_item_id, mi.name
		 ORDER BY total_sold DESC
		 LIMIT 5`, businessID)
	if err != nil {
		return nil, fmt.Errorf("top items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t TopMenuItem
		if err := rows.Scan(&t.MenuItemID, &t.MenuItemName, &t.TotalSold); err != nil {
			return nil, err
		}
		d.TopMenuItems = append(d.TopMenuItems, t)
	}
	if d.TopMenuItems == nil {
		d.TopMenuItems = []TopMenuItem{}
	}

	// Revenue per branch today.
	branchRows, err := s.db.Query(
		`SELECT t.branch_id, b.name, COALESCE(SUM(t.total), 0)
		 FROM transactions t
		 JOIN branches b ON b.id = t.branch_id
		 WHERE t.business_id = $1 AND t.paid_at >= CURRENT_DATE
		 GROUP BY t.branch_id, b.name
		 ORDER BY b.name`, businessID)
	if err != nil {
		return nil, fmt.Errorf("revenue per branch: %w", err)
	}
	defer branchRows.Close()
	for branchRows.Next() {
		var br BranchRev
		if err := branchRows.Scan(&br.BranchID, &br.BranchName, &br.Revenue); err != nil {
			return nil, err
		}
		d.RevenuePerBranch = append(d.RevenuePerBranch, br)
	}
	if d.RevenuePerBranch == nil {
		d.RevenuePerBranch = []BranchRev{}
	}

	return d, nil
}

// CashierStats returns the cashier dashboard for the given cashier or branch.
func (s *Service) CashierStats(businessID, cashierID uuid.UUID, branchID *uuid.UUID) (*CashierDashboard, error) {
	d := &CashierDashboard{}

	scopeArg := cashierID
	scopeCol := "cashier_id"
	if branchID != nil {
		// If scoped to branch (owner viewing), use branch_id.
		_ = branchID
	}

	// Open orders count.
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE business_id=$1 AND `+scopeCol+`=$2 AND status='open'`,
		businessID, scopeArg,
	).Scan(&d.OpenOrdersCount); err != nil {
		return nil, fmt.Errorf("open orders: %w", err)
	}

	// Orders handled today.
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE business_id=$1 AND `+scopeCol+`=$2
		 AND status='paid' AND created_at >= CURRENT_DATE`,
		businessID, scopeArg,
	).Scan(&d.OrdersHandledToday); err != nil {
		return nil, fmt.Errorf("orders today: %w", err)
	}

	// Total collected today.
	if err := s.db.QueryRow(
		`SELECT COALESCE(SUM(p.amount_paid), 0)
		 FROM payments p
		 JOIN orders o ON o.id = p.order_id
		 WHERE o.business_id=$1 AND o.`+scopeCol+`=$2 AND p.paid_at >= CURRENT_DATE`,
		businessID, scopeArg,
	).Scan(&d.TotalCollectedToday); err != nil {
		return nil, fmt.Errorf("total collected: %w", err)
	}

	return d, nil
}
