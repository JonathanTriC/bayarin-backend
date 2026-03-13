package payment

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the payment module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new payment handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Pay godoc
//
//	@Summary		Pay order
//	@Description	Process payment for an open order. Method must be "cash", "qris", or "transfer". amount_paid must be >= order total. Atomically marks order as paid and releases table if dine_in.
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string			true	"Order UUID"
//	@Param			body	body		PayOrderInput	true	"Payment payload"
//	@Success		201		{object}	Payment
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders/{id}/pay [post]
func (h *Handler) Pay(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	var input PayOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	p, err := h.svc.Pay(auth.BusinessID, orderID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": p})
}
