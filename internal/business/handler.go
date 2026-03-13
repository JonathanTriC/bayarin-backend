package business

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// Handler is the HTTP layer for the business module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new business handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Get godoc
//
//	@Summary		Get business
//	@Description	Returns the authenticated owner's business profile
//	@Tags			Business
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	Business
//	@Failure		404	{object}	httputil.ErrorResponse
//	@Router			/business [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	b, err := h.svc.Get(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": b})
}

// Update godoc
//
//	@Summary		Update business
//	@Description	Partially update business name, tax percent, or service charge percent
//	@Tags			Business
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UpdateBusinessInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	Business
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/business [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input UpdateBusinessInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	b, err := h.svc.Update(auth.BusinessID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": b})
}
