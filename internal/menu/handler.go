package menu

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the menu module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new menu handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List menu items
//	@Description	Returns all menu items for the authenticated owner's business, ordered by category and name
//	@Tags			Menu
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		MenuItem
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/menu [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	items, err := h.svc.List(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

// Create godoc
//
//	@Summary		Create menu item
//	@Description	Create a new menu item under the authenticated owner's business
//	@Tags			Menu
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateMenuItemInput	true	"Menu item payload"
//	@Success		201		{object}	MenuItem
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/menu [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateMenuItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	m, err := h.svc.Create(auth.BusinessID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": m})
}

// Update godoc
//
//	@Summary		Update menu item
//	@Description	Partially update a menu item's name, description, price, category, or availability
//	@Tags			Menu
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Menu item UUID"
//	@Param			body	body		UpdateMenuItemInput		true	"Update payload (all fields optional)"
//	@Success		200		{object}	MenuItem
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/menu/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid menu item id"})
	}
	var input UpdateMenuItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	m, err := h.svc.Update(auth.BusinessID, itemID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": m})
}
