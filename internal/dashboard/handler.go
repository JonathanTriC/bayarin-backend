package dashboard

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// Handler is the HTTP layer for the dashboard module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new dashboard handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// OwnerDashboard godoc
//
//	@Summary		Owner dashboard
//	@Description	Returns business-level stats: total revenue, orders, top menu items, and per-branch breakdown
//	@Tags			Dashboard
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	OwnerDashboard
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/dashboard/owner [get]
func (h *Handler) OwnerDashboard(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	stats, err := h.svc.OwnerStats(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": stats})
}

// CashierDashboard godoc
//
//	@Summary		Cashier dashboard
//	@Description	Returns cashier-level stats: today's transactions, open orders, and revenue for this session
//	@Tags			Dashboard
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	CashierDashboard
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/dashboard/cashier [get]
func (h *Handler) CashierDashboard(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	stats, err := h.svc.CashierStats(auth.BusinessID, auth.UserID, auth.BranchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": stats})
}
