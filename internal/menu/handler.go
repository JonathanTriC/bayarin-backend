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
//	@Description	List all menu items with nested modifier groups and options. Accessible by owner and cashier.
//	@Tags			menu
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		MenuItemResponse
//	@Failure		401	{object}	httputil.Error401Response
//	@Failure		500	{object}	httputil.Error500Response
//	@Router			/menu [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	items, err := h.svc.List(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

// Search godoc
//
//	@Summary		Search menu items
//	@Description	Search menu items by name or description (case-insensitive). Optionally filter by category. Accessible by owner and cashier.
//	@Tags			menu
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q			query		string	false	"Search keyword (matches name or description)"
//	@Param			category	query		string	false	"Filter by category"
//	@Success		200			{array}		MenuItemResponse
//	@Failure		401			{object}	httputil.Error401Response
//	@Failure		500			{object}	httputil.Error500Response
//	@Router			/menu/search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	items, err := h.svc.Search(auth.BusinessID, c.Query("q"), c.Query("category"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

// Create godoc
//
//	@Summary		Create menu item
//	@Description	Create a new menu item and optionally link modifier groups
//	@Tags			menu
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateMenuItemInput	true	"Menu item payload"
//	@Success		201		{object}	MenuItemResponse
//	@Failure 400 {object} httputil.Error400Response
//	@Failure 401 {object} httputil.Error401Response
//	@Failure 403 {object} httputil.Error403Response
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
//	@Description	Update menu item fields and replace modifier group links
//	@Tags			menu
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Menu item UUID"
//	@Param			body	body		UpdateMenuItemInput	true	"Update payload — modifier_group_ids replaces all existing links"
//	@Success		200		{object}	MenuItemResponse
//	@Failure 400 {object} httputil.Error400Response
//	@Failure 401 {object} httputil.Error401Response
//	@Failure 403 {object} httputil.Error403Response
//	@Failure 404 {object} httputil.Error404Response
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
