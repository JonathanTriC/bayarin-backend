package modifier

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the modifier module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new modifier handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List modifier groups
//	@Description	Returns all modifier groups (with their options) for the authenticated owner's business
//	@Tags			Modifiers
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		ModifierGroup
//	@Failure		500	{object}	httputil.Error500Response
//	@Router			/modifiers [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	groups, err := h.svc.List(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": groups})
}

// Search godoc
//
//	@Summary		Search modifier groups
//	@Description	Search modifier groups by name (case-insensitive).
//	@Tags			Modifiers
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q	query		string	false	"Search keyword (matches modifier group name)"
//	@Success		200	{array}		ModifierGroup
//	@Failure		401	{object}	httputil.Error401Response
//	@Failure		403	{object}	httputil.Error403Response
//	@Failure		500	{object}	httputil.Error500Response
//	@Router			/modifiers/search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	groups, err := h.svc.Search(auth.BusinessID, c.Query("q"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": groups})
}

// Create godoc
//
//	@Summary		Create modifier group
//	@Description	Create a modifier group with options in a single transaction
//	@Tags			Modifiers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateModifierGroupInput	true	"Modifier group payload"
//	@Success		201		{object}	ModifierGroup
//	@Failure 400 {object} httputil.Error400Response
//	@Router			/modifiers [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateModifierGroupInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	g, err := h.svc.Create(auth.BusinessID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": g})
}

// Update godoc
//
//	@Summary		Update modifier group
//	@Description	Partially update a modifier group's name, required flag, or max selections
//	@Tags			Modifiers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"Modifier group UUID"
//	@Param			body	body		UpdateModifierGroupInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	ModifierGroup
//	@Failure 400 {object} httputil.Error400Response
//	@Router			/modifiers/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	groupID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid modifier group id"})
	}
	var input UpdateModifierGroupInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	g, err := h.svc.Update(auth.BusinessID, groupID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": g})
}
