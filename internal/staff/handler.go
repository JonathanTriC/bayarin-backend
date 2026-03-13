package staff

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the staff module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new staff handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List staff
//	@Description	Returns all staff members for the authenticated owner's business
//	@Tags			Staff
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		Staff
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/staff [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	staff, err := h.svc.List(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": staff})
}

// Create godoc
//
//	@Summary		Create staff (cashier)
//	@Description	Create a new cashier account. Role is always set to "cashier". branch_id is required.
//	@Tags			Staff
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateStaffInput	true	"Staff payload"
//	@Success		201		{object}	Staff
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/staff [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateStaffInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	input.Role = "cashier"
	st, err := h.svc.Create(auth.BusinessID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": st})
}

// Update godoc
//
//	@Summary		Update staff
//	@Description	Update a staff member's name, active status, or assigned branch. Cannot deactivate owner.
//	@Tags			Staff
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Staff UUID"
//	@Param			body	body		UpdateStaffInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	Staff
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/staff/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	staffID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid staff id"})
	}
	var input UpdateStaffInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	st, err := h.svc.Update(auth.BusinessID, staffID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": st})
}
