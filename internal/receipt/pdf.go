package receipt

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/shopspring/decimal"
)

// GeneratePDF creates a styled PDF receipt scaled to an 80mm width standard.
func GeneratePDF(data *ReceiptData) ([]byte, error) {
	// A standard thermal roll is 80mm wide. We will set a generic long height (e.g. 297mm)
	// and FPDF will flow down automatically.
	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "mm",
		Size:    fpdf.SizeType{Wd: 80, Ht: 297},
	})

	pdf.SetMargins(5, 5, 5) // 5mm margins
	pdf.AddPage()

	// 1. Fetch & Embed Logo (Best Effort)
	if data.BusinessLogoURL != nil && *data.BusinessLogoURL != "" {
		func() {
			url := *data.BusinessLogoURL
			resp, err := http.Get(url)
			if err != nil {
				return // silently skip logo
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return
			}

			imgData, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			ext := ""
			if strings.HasSuffix(strings.ToLower(url), ".png") {
				ext = "png"
			} else if strings.HasSuffix(strings.ToLower(url), ".jpg") || strings.HasSuffix(strings.ToLower(url), ".jpeg") {
				ext = "jpg"
			}

			if ext != "" {
				imgOpt := fpdf.ImageOptions{ImageType: ext}
				pdf.RegisterImageOptionsReader(url, imgOpt, bytes.NewReader(imgData))
				
				// Center the image (say max width 30mm)
				imgWidth := 30.0
				x := (80.0 - imgWidth) / 2.0
				pdf.ImageOptions(url, x, pdf.GetY(), imgWidth, 0, false, imgOpt, 0, "")
				pdf.Ln(4) // add some space after image
			}
		}()
	}

	// 2. Business & Branch Info
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(70, 6, data.BusinessName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(70, 5, data.BranchName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(70, 4, data.BranchAddress, "", 1, "C", false, 0, "")

	pdf.Ln(2)
	drawLine(pdf)

	// 3. Order Metadata
	pdf.SetFont("Courier", "", 9)
	pdf.CellFormat(70, 4, fmt.Sprintf("Order   : %s", data.OrderNumber), "", 1, "L", false, 0, "")
	pdf.CellFormat(70, 4, fmt.Sprintf("Type    : %s", data.OrderType), "", 1, "L", false, 0, "")
	pdf.CellFormat(70, 4, fmt.Sprintf("Customer: %s", data.CustomerName), "", 1, "L", false, 0, "")
	pdf.CellFormat(70, 4, fmt.Sprintf("Cashier : %s", data.CashierName), "", 1, "L", false, 0, "")
	dateStr := data.OrderedAt.Format("02 Jan 2006 15:04")
	pdf.CellFormat(70, 4, fmt.Sprintf("Date    : %s", dateStr), "", 1, "L", false, 0, "")

	drawLine(pdf)

	// 4. Items Table (Qty | Item | Price)
	// Column X positions and widths (in mm, for 80mm paper)
	const (
		colItemX      = 5.0   // left margin
		colItemWidth  = 38.0  // ~55% of usable width
		colQtyX       = 43.0  // after item col
		colQtyWidth   = 10.0  // ~15%
		colPriceX     = 53.0  // after qty col
		colPriceWidth = 22.0  // ~30%, right-aligned
	)

	pdf.SetFont("Courier", "B", 9)
	pdf.SetX(colItemX)
	pdf.CellFormat(colItemWidth, 4, "ITEM", "", 0, "L", false, 0, "")
	pdf.SetX(colQtyX)
	pdf.CellFormat(colQtyWidth, 4, "QTY", "", 0, "C", false, 0, "")
	pdf.SetX(colPriceX)
	pdf.CellFormat(colPriceWidth, 4, "PRICE", "", 1, "R", false, 0, "")

	pdf.SetFont("Courier", "", 9)
	for _, item := range data.Items {
		// --- ITEM NAME ---
		lines := pdf.SplitLines([]byte(item.Name), colItemWidth)
		var firstLine, secondLine string
		if len(lines) > 0 {
			firstLine = string(lines[0])
		}
		if len(lines) > 1 {
			remainder := strings.TrimSpace(item.Name[len(firstLine):])
			if remainder != "" {
				l2 := pdf.SplitLines([]byte("  "+remainder), colItemWidth)
				if len(l2) > 0 {
					secondLine = string(l2[0])
					if len(l2) > 1 {
						// truncate with "..." if still too long
						secondLine += "..."
					}
				}
			}
		}

		pdf.SetX(colItemX)
		pdf.CellFormat(colItemWidth, 4, firstLine, "", 0, "L", false, 0, "")
		pdf.SetX(colQtyX)
		pdf.CellFormat(colQtyWidth, 4, fmt.Sprintf("%d", item.Quantity), "", 0, "C", false, 0, "")
		pdf.SetX(colPriceX)
		pdf.CellFormat(colPriceWidth, 4, formatRupiah(item.Subtotal), "", 1, "R", false, 0, "")

		if secondLine != "" {
			pdf.SetX(colItemX)
			pdf.CellFormat(colItemWidth, 4, secondLine, "", 1, "L", false, 0, "")
		}

		// --- MODIFIERS ---
		for _, mod := range item.Modifiers {
			pText := ""
			if mod.ExtraPrice.GreaterThan(decimal.NewFromInt(0)) {
				pText = formatRupiah(mod.ExtraPrice)
			}
			
			modText := "  + " + mod.Name
			modLines := pdf.SplitLines([]byte(modText), colItemWidth)
			var firstMod, secondMod string
			if len(modLines) > 0 {
				firstMod = string(modLines[0])
			}
			if len(modLines) > 1 {
				remainder := strings.TrimSpace(modText[len(firstMod):])
				if remainder != "" {
					l2 := pdf.SplitLines([]byte("    "+remainder), colItemWidth)
					if len(l2) > 0 {
						secondMod = string(l2[0])
						if len(l2) > 1 {
							secondMod += "..."
						}
					}
				}
			}

			pdf.SetX(colItemX)
			pdf.CellFormat(colItemWidth, 4, firstMod, "", 0, "L", false, 0, "")
			pdf.SetX(colQtyX)
			pdf.CellFormat(colQtyWidth, 4, "", "", 0, "C", false, 0, "")
			pdf.SetX(colPriceX)
			pdf.CellFormat(colPriceWidth, 4, pText, "", 1, "R", false, 0, "")

			if secondMod != "" {
				pdf.SetX(colItemX)
				pdf.CellFormat(colItemWidth, 4, secondMod, "", 1, "L", false, 0, "")
			}
		}

		// --- NOTES ---
		if item.Notes != "" {
			noteText := "  * " + item.Notes
			noteLines := pdf.SplitLines([]byte(noteText), colItemWidth)
			var firstNote, secondNote string
			if len(noteLines) > 0 {
				firstNote = string(noteLines[0])
			}
			if len(noteLines) > 1 {
				remainder := strings.TrimSpace(noteText[len(firstNote):])
				if remainder != "" {
					l2 := pdf.SplitLines([]byte("    "+remainder), colItemWidth)
					if len(l2) > 0 {
						secondNote = string(l2[0])
						if len(l2) > 1 {
							secondNote += "..."
						}
					}
				}
			}

			pdf.SetX(colItemX)
			pdf.CellFormat(colItemWidth, 4, firstNote, "", 1, "L", false, 0, "")
			if secondNote != "" {
				pdf.SetX(colItemX)
				pdf.CellFormat(colItemWidth, 4, secondNote, "", 1, "L", false, 0, "")
			}
		}
	}

	drawLine(pdf)

	// 5. Totals
	pdf.CellFormat(40, 4, "Subtotal           :", "", 0, "L", false, 0, "")
	pdf.CellFormat(30, 4, formatRupiah(data.Subtotal), "", 1, "R", false, 0, "")

	if data.TaxAmount.GreaterThan(decimal.NewFromInt(0)) {
		pdf.CellFormat(40, 4, fmt.Sprintf("Tax (%s%%)          :", data.TaxPercent.String()), "", 0, "L", false, 0, "")
		pdf.CellFormat(30, 4, formatRupiah(data.TaxAmount), "", 1, "R", false, 0, "")
	}

	if data.ServiceChargeAmount.GreaterThan(decimal.NewFromInt(0)) {
		pdf.CellFormat(40, 4, fmt.Sprintf("Service (%s%%):", data.ServiceChargePercent.String()), "", 0, "L", false, 0, "")
		pdf.CellFormat(30, 4, formatRupiah(data.ServiceChargeAmount), "", 1, "R", false, 0, "")
	}

	drawLine(pdf)

	pdf.SetFont("Courier", "B", 10)
	pdf.CellFormat(40, 5, "TOTAL              :", "", 0, "L", false, 0, "")
	pdf.CellFormat(30, 5, formatRupiah(data.Total), "", 1, "R", false, 0, "")

	pdf.SetFont("Courier", "", 9)
	drawLine(pdf)

	// 6. Payment
	pdf.CellFormat(70, 4, fmt.Sprintf("Payment : %s", data.PaymentMethod), "", 1, "L", false, 0, "")
	
	pdf.CellFormat(20, 4, "Paid    :", "", 0, "L", false, 0, "")
	pdf.CellFormat(50, 4, formatRupiah(data.AmountPaid), "", 1, "L", false, 0, "") // Matching prompt layout spacing roughly

	pdf.CellFormat(20, 4, "Change  :", "", 0, "L", false, 0, "")
	pdf.CellFormat(50, 4, formatRupiah(data.ChangeAmount), "", 1, "L", false, 0, "")

	drawLine(pdf)

	// 7. Footer
	pdf.SetFont("Arial", "I", 9)
	pdf.Ln(2)
	pdf.CellFormat(70, 4, "Thank you!", "", 1, "C", false, 0, "")
	pdf.CellFormat(70, 4, "Powered by Bayar.in", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawLine(pdf *fpdf.Fpdf) {
	pdf.CellFormat(70, 4, "--------------------------------", "", 1, "C", false, 0, "")
}
