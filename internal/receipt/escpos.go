package receipt

import (
	"bytes"
	"fmt"

	"github.com/shopspring/decimal"
)

const (
	escInit        = "\x1B\x40"
	escAlignLeft   = "\x1B\x61\x00"
	escAlignCenter = "\x1B\x61\x01"
	escAlignRight  = "\x1B\x61\x02"
	escBoldOn      = "\x1B\x45\x01"
	escBoldOff     = "\x1B\x45\x00"
	escFontNormal  = "\x1B\x21\x00"
	escFontLarge   = "\x1B\x21\x30"
	escLF          = "\x0A"
	escCut         = "\x1D\x56\x42\x00"
	lineWidth      = 48
)

// GenerateESCPOS creates raw ESC/POS byte commands for an 80mm thermal printer.
func GenerateESCPOS(data *ReceiptData) ([]byte, error) {
	var buf bytes.Buffer

	// Initialization
	buf.WriteString(escInit)

	// Business Info
	buf.WriteString(escAlignCenter)
	buf.WriteString(escBoldOn)
	buf.WriteString(escFontLarge)
	buf.WriteString(data.BusinessName)
	buf.WriteString(escLF)
	buf.WriteString(escFontNormal)
	buf.WriteString(escBoldOff)

	buf.WriteString(data.BranchName)
	buf.WriteString(escLF)
	buf.WriteString(data.BranchAddress)
	buf.WriteString(escLF)

	// Divider
	buf.WriteString(escAlignLeft)
	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// Order Info
	buf.WriteString(fmt.Sprintf("Order   : %s\n", data.OrderNumber))
	buf.WriteString(fmt.Sprintf("Type    : %s\n", data.OrderType))
	buf.WriteString(fmt.Sprintf("Customer: %s\n", data.CustomerName))
	buf.WriteString(fmt.Sprintf("Cashier : %s\n", data.CashierName))
	// Date formatting
	dateStr := data.OrderedAt.Format("02 Jan 2006 15:04")
	buf.WriteString(fmt.Sprintf("Date    : %s\n", dateStr))

	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// Items
	for _, item := range data.Items {
		left := fmt.Sprintf("%s x%d", item.Name, item.Quantity)
		right := formatRupiah(item.Subtotal)
		buf.WriteString(formatLine(left, right, lineWidth))
		buf.WriteString(escLF)

		for _, mod := range item.Modifiers {
			mLeft := fmt.Sprintf("  + %s", mod.Name)
			var mRight string
			if mod.ExtraPrice.GreaterThan(decimalZero()) {
				mRight = formatRupiah(mod.ExtraPrice)
			}
			buf.WriteString(formatLine(mLeft, mRight, lineWidth))
			buf.WriteString(escLF)
		}

		if item.Notes != "" {
			buf.WriteString(fmt.Sprintf("  * %s\n", item.Notes))
		}
	}

	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// Totals
	buf.WriteString(formatLine("Subtotal", formatRupiah(data.Subtotal), lineWidth))
	buf.WriteString(escLF)
	if data.TaxAmount.GreaterThan(decimalZero()) {
		taxLabel := fmt.Sprintf("Tax (%s%%)", data.TaxPercent.String())
		buf.WriteString(formatLine(taxLabel, formatRupiah(data.TaxAmount), lineWidth))
		buf.WriteString(escLF)
	}
	if data.ServiceChargeAmount.GreaterThan(decimalZero()) {
		svcLabel := fmt.Sprintf("Service (%s%%)", data.ServiceChargePercent.String())
		buf.WriteString(formatLine(svcLabel, formatRupiah(data.ServiceChargeAmount), lineWidth))
		buf.WriteString(escLF)
	}

	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// GRAND TOTAL
	buf.WriteString(escBoldOn)
	buf.WriteString(escFontLarge)
	buf.WriteString(formatLine("TOTAL", formatRupiah(data.Total), lineWidth))
	buf.WriteString(escLF)
	buf.WriteString(escFontNormal)
	buf.WriteString(escBoldOff)

	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// Payment Info
	buf.WriteString(fmt.Sprintf("Payment : %s\n", data.PaymentMethod))
	buf.WriteString(formatLine("Paid", formatRupiah(data.AmountPaid), lineWidth))
	buf.WriteString(escLF)
	buf.WriteString(formatLine("Change", formatRupiah(data.ChangeAmount), lineWidth))
	buf.WriteString(escLF)

	buf.WriteString(divider(lineWidth))
	buf.WriteString(escLF)

	// Footer
	buf.WriteString(escAlignCenter)
	buf.WriteString("Thank you!\n")
	buf.WriteString("Powered by Bayar.in\n")
	buf.WriteString(escLF)
	buf.WriteString(escLF)
	buf.WriteString(escLF)

	// Cut
	buf.WriteString(escCut)

	return buf.Bytes(), nil
}

func decimalZero() decimal.Decimal {
	return decimal.NewFromInt(0)
}
