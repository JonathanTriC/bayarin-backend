package receipt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bayarin/backend/internal/db/sqlcgen"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service interface {
	GetReceiptData(ctx context.Context, orderID uuid.UUID, businessID uuid.UUID) (*ReceiptData, error)
	GeneratePDF(data *ReceiptData) ([]byte, error)
	GenerateESCPOS(data *ReceiptData) ([]byte, error)
}

type serviceImpl struct {
	db *sql.DB
}

// NewService creates a new receipt service.
func NewService(db *sql.DB) Service {
	return &serviceImpl{db: db}
}

func (s *serviceImpl) GetReceiptData(ctx context.Context, orderID uuid.UUID, businessID uuid.UUID) (*ReceiptData, error) {
	q := sqlcgen.New(s.db)

	// 1. Base Data
	base, err := q.GetReceiptData(ctx, sqlcgen.GetReceiptDataParams{
		ID:         orderID,
		BusinessID: businessID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found or unauthorized")
		}
		return nil, fmt.Errorf("fetch base order data: %w", err)
	}

	// 2. Validate Paid Status
	if base.OrderStatus != "paid" {
		return nil, errors.New("receipt only available for paid orders")
	}

	// 3. Get Items
	itemsRow, err := q.GetReceiptItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("fetch order items: %w", err)
	}

	// 4. Batch fetch modifiers
	itemIDs := make([]uuid.UUID, len(itemsRow))
	for i, item := range itemsRow {
		itemIDs[i] = item.ItemID
	}

	modsRow, err := q.GetReceiptItemModifiers(ctx, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch item modifiers: %w", err)
	}

	// 5. Map modifiers back to items
	modsMap := make(map[uuid.UUID][]ReceiptModifier)
	for _, m := range modsRow {
		priceDec, _ := decimal.NewFromString(fmt.Sprintf("%v", m.ExtraPrice))
		modsMap[m.OrderItemID] = append(modsMap[m.OrderItemID], ReceiptModifier{
			Name:       m.ModifierName,
			ExtraPrice: priceDec,
		})
	}

	parsedItems := make([]ReceiptItem, len(itemsRow))
	for i, row := range itemsRow {
		unitDec, _ := decimal.NewFromString(fmt.Sprintf("%v", row.UnitPrice))
		subDec, _ := decimal.NewFromString(fmt.Sprintf("%v", row.Subtotal))
		parsedItems[i] = ReceiptItem{
			Name:      row.ItemName,
			Quantity:  int(row.Quantity),
			UnitPrice: unitDec,
			Subtotal:  subDec,
			Notes:     row.Notes,
			Modifiers: modsMap[row.ItemID],
		}
	}

	// 6. Map payment method and order type presentation
	paymentMethod := "Cash"
	switch base.PaymentMethod {
	case "qris":
		paymentMethod = "QRIS"
	case "transfer":
		paymentMethod = "Transfer"
	}

	orderType := "Takeaway"
	if base.OrderType == "dine_in" {
		orderType = "Dine In"
	}

	var logoURL *string
	if base.BusinessLogoUrl.Valid {
		logoURL = &base.BusinessLogoUrl.String
	}

	// 7. Parse numerics for totals
	subtotalDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.Subtotal))
	taxAmountDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.TaxAmount))
	taxPercentDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.TaxPercent))
	svcAmountDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.ServiceChargeAmount))
	svcPercentDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.ServiceChargePercent))
	totalDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.Total))
	amountPaidDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.AmountPaid))
	changeAmountDec, _ := decimal.NewFromString(fmt.Sprintf("%v", base.ChangeAmount))

	return &ReceiptData{
		BusinessName:         base.BusinessName,
		BusinessLogoURL:      logoURL,
		BranchName:           base.BranchName,
		BranchAddress:        base.BranchAddress,
		OrderID:              base.OrderID,
		OrderNumber:          base.OrderNumber.String,
		OrderType:            orderType,
		CustomerName:         base.CustomerName,
		OrderedAt:            base.OrderedAt,
		CashierName:          base.CashierName,
		Items:                parsedItems,
		Subtotal:             subtotalDec,
		TaxPercent:           taxPercentDec,
		TaxAmount:            taxAmountDec,
		ServiceChargePercent: svcPercentDec,
		ServiceChargeAmount:  svcAmountDec,
		Total:                totalDec,
		PaymentMethod:        paymentMethod,
		AmountPaid:           amountPaidDec,
		ChangeAmount:         changeAmountDec,
		PaidAt:               base.PaidAt,
	}, nil
}

func (s *serviceImpl) GeneratePDF(data *ReceiptData) ([]byte, error) {
	return GeneratePDF(data)
}

func (s *serviceImpl) GenerateESCPOS(data *ReceiptData) ([]byte, error) {
	return GenerateESCPOS(data)
}
