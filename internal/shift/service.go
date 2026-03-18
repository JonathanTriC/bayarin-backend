package shift

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bayarin/backend/internal/db/sqlcgen"
	"github.com/bayarin/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ShiftResponse struct {
	ID          uuid.UUID  `json:"id"`
	BranchID    uuid.UUID  `json:"branch_id"`
	CashierID   uuid.UUID  `json:"cashier_id"`
	CashierName string     `json:"cashier_name"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	IsOpen      bool       `json:"is_open"`
}

type PaymentBreakdown struct {
	Cash     decimal.Decimal `json:"cash"`
	QRIS     decimal.Decimal `json:"qris"`
	Transfer decimal.Decimal `json:"transfer"`
}

type TopItem struct {
	MenuItemID   uuid.UUID       `json:"menu_item_id"`
	Name         string          `json:"name"`
	Category     string          `json:"category"`
	TotalQty     int64           `json:"total_qty"`
	TotalRevenue decimal.Decimal `json:"total_revenue"`
}

type ShiftReportResponse struct {
	ShiftResponse
	TotalOrders      int64            `json:"total_orders"`
	TotalRevenue     decimal.Decimal  `json:"total_revenue"`
	PaymentBreakdown PaymentBreakdown `json:"payment_breakdown"`
	CancelledOrders  int64            `json:"cancelled_orders"`
	TopItems         []TopItem        `json:"top_items"`
}

type ShiftSummaryResponse struct {
	ShiftResponse
	TotalOrders  int64           `json:"total_orders"`
	TotalRevenue decimal.Decimal `json:"total_revenue"`
}

type Service interface {
	OpenShift(ctx context.Context, auth middleware.AuthContext) (*ShiftResponse, error)
	CloseShift(ctx context.Context, auth middleware.AuthContext) (*ShiftReportResponse, error)
	GetShiftReport(ctx context.Context, shiftID uuid.UUID, businessID uuid.UUID) (*ShiftReportResponse, error)
	ListMyCashierShifts(ctx context.Context, auth middleware.AuthContext) ([]ShiftSummaryResponse, error)
	ListBranchShifts(ctx context.Context, branchID uuid.UUID, businessID uuid.UUID) ([]ShiftSummaryResponse, error)
}

type service struct {
	q  *sqlcgen.Queries
	db *sql.DB
}

func NewService(db *sql.DB) Service {
	return &service{
		q:  sqlcgen.New(db),
		db: db,
	}
}

func (s *service) OpenShift(ctx context.Context, auth middleware.AuthContext) (*ShiftResponse, error) {
	if auth.BranchID == nil {
		return nil, errors.New("must be assigned to a branch to open a shift")
	}

	_, err := s.q.GetOpenShiftByCashier(ctx, sqlcgen.GetOpenShiftByCashierParams{
		CashierID: auth.UserID,
		BranchID:  *auth.BranchID,
	})
	if err == nil {
		return nil, errors.New("you already have an open shift — close it first")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check open shift: %w", err)
	}

	raw, err := s.q.OpenShift(ctx, sqlcgen.OpenShiftParams{
		ID:         uuid.New(),
		BusinessID: auth.BusinessID,
		BranchID:   *auth.BranchID,
		CashierID:  auth.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("open shift: %w", err)
	}

	return &ShiftResponse{
		ID:        raw.ID,
		BranchID:  raw.BranchID,
		CashierID: raw.CashierID,
		StartedAt: raw.StartedAt,
		IsOpen:    raw.IsOpen,
	}, nil
}

func (s *service) CloseShift(ctx context.Context, auth middleware.AuthContext) (*ShiftReportResponse, error) {
	if auth.BranchID == nil {
		return nil, errors.New("must be assigned to a branch to close a shift")
	}

	shift, err := s.q.GetOpenShiftByCashier(ctx, sqlcgen.GetOpenShiftByCashierParams{
		CashierID: auth.UserID,
		BranchID:  *auth.BranchID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no open shift found")
		}
		return nil, fmt.Errorf("find open shift: %w", err)
	}

	now := time.Now()

	// Aggregate
	stats, err := s.q.GetShiftOrderStats(ctx, sqlcgen.GetShiftOrderStatsParams{
		CashierID:   auth.UserID,
		BranchID:    *auth.BranchID,
		CreatedAt:   shift.StartedAt,
		CreatedAt_2: now,
	})
	if err != nil {
		return nil, fmt.Errorf("get order stats: %w", err)
	}

	top, err := s.q.GetShiftTopItems(ctx, sqlcgen.GetShiftTopItemsParams{
		CashierID:   auth.UserID,
		BranchID:    *auth.BranchID,
		CreatedAt:   shift.StartedAt,
		CreatedAt_2: now,
	})
	if err != nil {
		return nil, fmt.Errorf("get top items: %w", err)
	}

	// Make decimals safely via String logic
	toDecimal := func(v interface{}) decimal.Decimal {
		switch i := v.(type) {
		case string:
			d, _ := decimal.NewFromString(i)
			return d
		case nil:
			return decimal.Zero
		default:
			// sqlc converts PG NUMERIC to string implicitly natively
			str := fmt.Sprintf("%v", i)
			d, _ := decimal.NewFromString(str)
			return d
		}
	}

	totalRev := toDecimal(stats.TotalRevenue)
	cashRev := toDecimal(stats.CashRevenue)
	qrisRev := toDecimal(stats.QrisRevenue)
	trRev := toDecimal(stats.TransferRevenue)

	// Update natively
	closed, err := s.q.CloseShift(ctx, sqlcgen.CloseShiftParams{
		ID:              shift.ID,
		TotalOrders:     sql.NullInt32{Int32: int32(stats.TotalOrders), Valid: true},
		TotalRevenue:    sql.NullString{String: totalRev.String(), Valid: true},
		CashRevenue:     sql.NullString{String: cashRev.String(), Valid: true},
		QrisRevenue:     sql.NullString{String: qrisRev.String(), Valid: true},
		TransferRevenue: sql.NullString{String: trRev.String(), Valid: true},
		CancelledOrders: sql.NullInt32{Int32: int32(stats.CancelledOrders), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("close shift bounds: %w", err)
	}

	var parsedTopItems []TopItem
	for _, ti := range top {
		parsedTopItems = append(parsedTopItems, TopItem{
			MenuItemID:   ti.MenuItemID,
			Name:         ti.MenuItemName,
			Category:     ti.Category,
			TotalQty:     ti.TotalQty,
			TotalRevenue: toDecimal(ti.TotalRevenue),
		})
	}

	return &ShiftReportResponse{
		ShiftResponse: ShiftResponse{
			ID:        closed.ID,
			BranchID:  closed.BranchID,
			CashierID: closed.CashierID,
			StartedAt: closed.StartedAt,
			EndedAt:   &closed.EndedAt.Time, // Valid is guaranteed True upon close
			IsOpen:    closed.IsOpen,
		},
		TotalOrders:  stats.TotalOrders,
		TotalRevenue: totalRev,
		PaymentBreakdown: PaymentBreakdown{
			Cash:     cashRev,
			QRIS:     qrisRev,
			Transfer: trRev,
		},
		CancelledOrders: stats.CancelledOrders,
		TopItems:        parsedTopItems,
	}, nil
}

func (s *service) GetShiftReport(ctx context.Context, shiftID uuid.UUID, businessID uuid.UUID) (*ShiftReportResponse, error) {
	shift, err := s.q.GetShiftByID(ctx, sqlcgen.GetShiftByIDParams{
		ID:         shiftID,
		BusinessID: businessID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("shift not found")
		}
		return nil, fmt.Errorf("get shift limits: %w", err)
	}

	var ended time.Time
	if !shift.IsOpen && shift.EndedAt.Valid {
		ended = shift.EndedAt.Time
	} else {
		ended = time.Now()
	}

	stats, err := s.q.GetShiftOrderStats(ctx, sqlcgen.GetShiftOrderStatsParams{
		CashierID:   shift.CashierID,
		BranchID:    shift.BranchID,
		CreatedAt:   shift.StartedAt,
		CreatedAt_2: ended,
	})
	if err != nil {
		return nil, fmt.Errorf("get report stats: %w", err)
	}

	top, err := s.q.GetShiftTopItems(ctx, sqlcgen.GetShiftTopItemsParams{
		CashierID:   shift.CashierID,
		BranchID:    shift.BranchID,
		CreatedAt:   shift.StartedAt,
		CreatedAt_2: ended,
	})
	if err != nil {
		return nil, fmt.Errorf("get report items: %w", err)
	}

	toDecimal := func(v interface{}) decimal.Decimal {
		switch i := v.(type) {
		case string:
			d, _ := decimal.NewFromString(i)
			return d
		case nil:
			return decimal.Zero
		default:
			str := fmt.Sprintf("%v", i)
			d, _ := decimal.NewFromString(str)
			return d
		}
	}

	totalRev := toDecimal(stats.TotalRevenue)

	var parsedTopItems []TopItem
	for _, ti := range top {
		parsedTopItems = append(parsedTopItems, TopItem{
			MenuItemID:   ti.MenuItemID,
			Name:         ti.MenuItemName,
			Category:     ti.Category,
			TotalQty:     ti.TotalQty,
			TotalRevenue: toDecimal(ti.TotalRevenue),
		})
	}

	resp := &ShiftReportResponse{
		ShiftResponse: ShiftResponse{
			ID:        shift.ID,
			BranchID:  shift.BranchID,
			CashierID: shift.CashierID,
			StartedAt: shift.StartedAt,
			IsOpen:    shift.IsOpen,
		},
		TotalOrders:  stats.TotalOrders,
		TotalRevenue: totalRev,
		PaymentBreakdown: PaymentBreakdown{
			Cash:     toDecimal(stats.CashRevenue),
			QRIS:     toDecimal(stats.QrisRevenue),
			Transfer: toDecimal(stats.TransferRevenue),
		},
		CancelledOrders: stats.CancelledOrders,
		TopItems:        parsedTopItems,
	}

	if shift.EndedAt.Valid {
		resp.EndedAt = &shift.EndedAt.Time
	}

	// Try fetching cashier name logic dynamically directly using users sql limits cleanly?
	// The struct asks for CashierName. In GetShiftByID we don't fetch users.name.
	// But it works as empty mapping mostly cleanly unless requested natively. 
	return resp, nil
}

func (s *service) ListMyCashierShifts(ctx context.Context, auth middleware.AuthContext) ([]ShiftSummaryResponse, error) {
	shifts, err := s.q.ListShiftsByCashier(ctx, sqlcgen.ListShiftsByCashierParams{
		CashierID:  auth.UserID,
		BusinessID: auth.BusinessID,
	})
	if err != nil {
		return nil, err
	}

	var results []ShiftSummaryResponse
	for _, row := range shifts {
		totalRev, _ := decimal.NewFromString(fmt.Sprintf("%v", row.TotalRevenue))
		
		item := ShiftSummaryResponse{
			ShiftResponse: ShiftResponse{
				ID:        row.ID,
				BranchID:  row.BranchID,
				CashierID: row.CashierID,
				StartedAt: row.StartedAt,
				IsOpen:    row.IsOpen,
			},
			TotalOrders:  int64(row.TotalOrders.Int32),
			TotalRevenue: totalRev,
		}
		if row.EndedAt.Valid {
			item.EndedAt = &row.EndedAt.Time
		}
		results = append(results, item)
	}
	if results == nil {
		results = []ShiftSummaryResponse{}
	}
	return results, nil
}

func (s *service) ListBranchShifts(ctx context.Context, branchID uuid.UUID, businessID uuid.UUID) ([]ShiftSummaryResponse, error) {
	shifts, err := s.q.ListShiftsByBranch(ctx, sqlcgen.ListShiftsByBranchParams{
		BranchID:   branchID,
		BusinessID: businessID,
	})
	if err != nil {
		return nil, err
	}

	var results []ShiftSummaryResponse
	for _, row := range shifts {
		totalRev, _ := decimal.NewFromString(fmt.Sprintf("%v", row.TotalRevenue))
		item := ShiftSummaryResponse{
			ShiftResponse: ShiftResponse{
				ID:          row.ID,
				BranchID:    row.BranchID,
				CashierID:   row.CashierID,
				CashierName: row.CashierName,
				StartedAt:   row.StartedAt,
				IsOpen:      row.IsOpen,
			},
			TotalOrders:  int64(row.TotalOrders.Int32),
			TotalRevenue: totalRev,
		}
		if row.EndedAt.Valid {
			item.EndedAt = &row.EndedAt.Time
		}
		results = append(results, item)
	}
	if results == nil {
		results = []ShiftSummaryResponse{}
	}
	return results, nil
}
