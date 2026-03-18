package receipt

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ReceiptData struct {
	// Business
	BusinessName    string
	BusinessLogoURL *string

	// Branch
	BranchName    string
	BranchAddress string

	// Order
	OrderID      uuid.UUID
	OrderNumber  string // e.g. "ORD-0042" from db sequence
	OrderType    string // "Dine In" or "Takeaway"
	CustomerName string
	OrderedAt    time.Time

	// Cashier
	CashierName string

	// Items
	Items []ReceiptItem

	// Totals
	Subtotal             decimal.Decimal
	TaxPercent           decimal.Decimal
	TaxAmount            decimal.Decimal
	ServiceChargePercent decimal.Decimal
	ServiceChargeAmount  decimal.Decimal
	Total                decimal.Decimal

	// Payment
	PaymentMethod string
	AmountPaid    decimal.Decimal
	ChangeAmount  decimal.Decimal
	PaidAt        time.Time
}

type ReceiptItem struct {
	Name      string
	Quantity  int
	UnitPrice decimal.Decimal
	Subtotal  decimal.Decimal
	Notes     string
	Modifiers []ReceiptModifier
}

type ReceiptModifier struct {
	Name       string
	ExtraPrice decimal.Decimal
}

// formatRupiah formats decimal as Indonesian Rupiah with "Rp." prefix and thousand separator
// e.g. 104400 → "Rp.104,400"
func formatRupiah(amount decimal.Decimal) string {
	s := amount.Truncate(0).String()
	isNegative := false
	if strings.HasPrefix(s, "-") {
		isNegative = true
		s = s[1:]
	}

	n := len(s)
	if n <= 3 {
		if isNegative {
			return "-Rp." + s
		}
		return "Rp." + s
	}

	var builder strings.Builder
	if isNegative {
		builder.WriteString("-")
	}
	builder.WriteString("Rp.")

	first := n % 3
	if first > 0 {
		builder.WriteString(s[:first])
		if n > first {
			builder.WriteString(",")
		}
	}
	for i := first; i < n; i += 3 {
		builder.WriteString(s[i : i+3])
		if i+3 < n {
			builder.WriteString(",")
		}
	}
	return builder.String()
}

// formatLine pads string to fill width with spaces between left and right
// e.g. formatLine("Subtotal", "Rp.90,000", 48) → "Subtotal                       Rp.90,000"
func formatLine(left, right string, width int) string {
	spaces := width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1 // force at least 1 space
	}
	return left + strings.Repeat(" ", spaces) + right
}

// divider returns a string of dashes of given width
func divider(width int) string {
	return strings.Repeat("-", width)
}
