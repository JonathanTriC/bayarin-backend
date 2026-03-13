package order

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the order module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new order handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List orders
//	@Description	Returns orders for the business. Cashiers see only their branch's orders. Filter by status: open|paid|cancelled.
//	@Tags			Orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"Filter by status (open, paid, cancelled)"
//	@Success		200		{array}		Order
//	@Failure		500		{object}	httputil.ErrorResponse
//	@Router			/orders [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orders, err := h.svc.List(auth, c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

// Create godoc
//
//	@Summary		Create order
//	@Description	Create a new order. Type must be "dine_in" or "takeaway". table_id is optional.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateOrderInput	true	"Order payload"
//	@Success		201		{object}	Order
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	o, err := h.svc.Create(auth, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": o})
}

// GetByID godoc
//
//	@Summary		Get order by ID
//	@Description	Returns a single order with all its items and applied modifiers
//	@Tags			Orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order UUID"
//	@Success		200	{object}	Order
//	@Failure		404	{object}	httputil.ErrorResponse
//	@Router			/orders/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	o, err := h.svc.GetByID(auth.BusinessID, orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": o})
}

// Update godoc
//
//	@Summary		Update order
//	@Description	Update order status or customer name. Cannot update paid or cancelled orders.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Order UUID"
//	@Param			body	body		UpdateOrderInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	Order
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	var input UpdateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	o, err := h.svc.Update(auth.BusinessID, orderID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": o})
}

// AddItem godoc
//
//	@Summary		Add item to order
//	@Description	Add a menu item (with optional modifier options) to an open order. Recalculates totals.
//	@Tags			Order Items
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Order UUID"
//	@Param			body	body		AddOrderItemInput	true	"Item payload"
//	@Success		201		{object}	OrderItem
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders/{id}/items [post]
func (h *Handler) AddItem(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	var input AddOrderItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	item, err := h.svc.AddItem(auth.BusinessID, orderID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

// UpdateItem godoc
//
//	@Summary		Update order item
//	@Description	Update quantity or notes of an existing order item. Only works on open orders.
//	@Tags			Order Items
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Order UUID"
//	@Param			item_id	path		string					true	"Order item UUID"
//	@Param			body	body		UpdateOrderItemInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	OrderItem
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders/{id}/items/{item_id} [patch]
func (h *Handler) UpdateItem(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	itemID, err := uuid.Parse(c.Params("item_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid item id"})
	}
	var input UpdateOrderItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	item, err := h.svc.UpdateItem(auth.BusinessID, orderID, itemID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

// DeleteItem godoc
//
//	@Summary		Delete order item
//	@Description	Remove an item from an open order. Recalculates order totals.
//	@Tags			Order Items
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Order UUID"
//	@Param			item_id	path		string	true	"Order item UUID"
//	@Success		200		{object}	httputil.MessageResponse
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/orders/{id}/items/{item_id} [delete]
func (h *Handler) DeleteItem(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid order id"})
	}
	itemID, err := uuid.Parse(c.Params("item_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid item id"})
	}
	if err := h.svc.DeleteItem(auth.BusinessID, orderID, itemID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": "item deleted"})
}
