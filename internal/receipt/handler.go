package receipt

import (
	"fmt"

	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler handles receipt HTTP requests.
type Handler struct {
	svc Service
}

// NewHandler creates a new receipt handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// GetReceiptData godoc
//
//	@Summary		Get receipt data
//	@Description	Returns full receipt data for a paid order as JSON
//	@Tags			receipt
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{object}	map[string]interface{}	"Receipt data"
//	@Failure		400	{object}	map[string]interface{}	"Order not paid yet"
//	@Failure		404	{object}	map[string]interface{}	"Order not found"
//	@Router			/orders/{id}/receipt [get]
func (h *Handler) GetReceiptData(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}

	data, err := h.svc.GetReceiptData(c.Context(), orderID, auth.BusinessID)
	if err != nil {
		if err.Error() == "order not found or unauthorized" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": data})
}

// DownloadPDF godoc
//
//	@Summary		Download receipt PDF
//	@Description	Generates and returns a PDF receipt for a paid order
//	@Tags			receipt
//	@Produce		application/pdf
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{file}		binary	"PDF receipt file"
//	@Failure		404	{object}	map[string]interface{}	"Order not found"
//	@Router			/orders/{id}/receipt/pdf [get]
func (h *Handler) DownloadPDF(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}

	data, err := h.svc.GetReceiptData(c.Context(), orderID, auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	pdfBytes, err := h.svc.GeneratePDF(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "failed to generate PDF: " + err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=receipt-%s.pdf", orderID.String()))
	return c.Send(pdfBytes)
}

// DownloadESCPOS godoc
//
//	@Summary		Get ESC/POS receipt bytes
//	@Description	Returns raw ESC/POS byte stream for direct thermal printer sending
//	@Tags			receipt
//	@Produce		application/octet-stream
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{file}		binary	"Raw ESC/POS bytes"
//	@Failure		404	{object}	map[string]interface{}	"Order not found"
//	@Router			/orders/{id}/receipt/escpos [get]
func (h *Handler) DownloadESCPOS(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}

	data, err := h.svc.GetReceiptData(c.Context(), orderID, auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	escposBytes, err := h.svc.GenerateESCPOS(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "failed to generate escpos: " + err.Error()})
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=receipt-%s.bin", orderID.String()))
	return c.Send(escposBytes)
}
