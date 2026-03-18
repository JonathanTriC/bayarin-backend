package shift

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// OpenShift godoc
//
//	@Summary		Open shift
//	@Description	Cashier opens a new shift. Only one active shift allowed per cashier per branch.
//	@Tags			shifts
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	map[string]interface{}	"Shift opened"
//	@Failure		400	{object}	map[string]interface{}	"Already have an open shift"
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Router			/shifts/open [post]
func (h *Handler) OpenShift(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	resp, err := h.svc.OpenShift(c.Context(), auth)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": resp})
}

// CloseShift godoc
//
//	@Summary		Close shift
//	@Description	Cashier closes their active shift. Returns full shift report with stats.
//	@Tags			shifts
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"Shift closed with report"
//	@Failure		400	{object}	map[string]interface{}	"No open shift found"
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Router			/shifts/close [post]
func (h *Handler) CloseShift(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	resp, err := h.svc.CloseShift(c.Context(), auth)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": resp})
}

// ListMyShifts godoc
//
//	@Summary		List my shifts
//	@Description	Returns the last 50 shifts for the authenticated cashier
//	@Tags			shifts
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"List of shifts"
//	@Router			/shifts/my [get]
func (h *Handler) ListMyShifts(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	resp, err := h.svc.ListMyCashierShifts(c.Context(), auth)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": resp})
}

// GetShiftReport godoc
//
//	@Summary		Get shift report
//	@Description	Returns full report for a specific shift including stats and top items
//	@Tags			shifts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Shift UUID"
//	@Success		200	{object}	map[string]interface{}	"Shift report"
//	@Failure		404	{object}	map[string]interface{}	"Shift not found"
//	@Router			/shifts/{id}/report [get]
func (h *Handler) GetShiftReport(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid shift id"})
	}

	resp, err := h.svc.GetShiftReport(c.Context(), id, auth.BusinessID)
	if err != nil {
		if err.Error() == "shift not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": resp})
}

// ListBranchShifts godoc
//
//	@Summary		List branch shifts
//	@Description	Owner views all shifts across all cashiers for a branch (last 100)
//	@Tags			shifts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			branch_id	query	string	true	"Branch UUID"
//	@Success		200	{object}	map[string]interface{}	"List of branch shifts"
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Failure		403	{object}	map[string]interface{}	"Forbidden - owner only"
//	@Router			/shifts/branch [get]
func (h *Handler) ListBranchShifts(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	branchID, err := uuid.Parse(c.Query("branch_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch_id query parameter"})
	}

	resp, err := h.svc.ListBranchShifts(c.Context(), branchID, auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": resp})
}
