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
//	@Failure 500 {object} httputil.Error500Response
//	@Router			/orders [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	
	// If cashier → automatically filter by auth.BranchID
	// If owner → allow branch_id as optional query param (if provided)
	// We handle this inside service natively, but we can pass query gracefully.
	
	orders, err := h.svc.List(auth, c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

// Search godoc
//
//	@Summary		Search orders
//	@Description	Search orders by customer name (case-insensitive). Optionally filter by type (dine_in|takeaway) and status (open|paid|cancelled). Cashiers are scoped to their branch.
//	@Tags			Orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q		query		string	false	"Search keyword (matches customer name)"
//	@Param			type	query		string	false	"Filter by order type: dine_in or takeaway"
//	@Param			status	query		string	false	"Filter by status: open, paid, or cancelled"
//	@Success		200		{array}		Order
//	@Failure		401		{object}	httputil.Error401Response
//	@Failure		403		{object}	httputil.Error403Response
//	@Failure		500		{object}	httputil.Error500Response
//	@Router			/orders/search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	orders, err := h.svc.Search(auth, c.Query("q"), c.Query("type"), c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

// Create godoc
//
//	@Summary		Create order
//	@Description	Create a new order. Type must be "dine_in" or "takeaway". table_id is optional. Owners must provide branch_id query.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			branch_id	query		string				false	"Branch UUID (Required for Owners)"
//	@Param			body		body		CreateOrderInput	true	"Order payload"
//	@Success		201		{object}	Order
//	@Failure 400 {object} httputil.Error400Response
//	@Router			/orders [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)

	var payload struct {
		BranchID     string `json:"branch_id"`
		TableID      string `json:"table_id"`
		Type         string `json:"type"`
		CustomerName string `json:"customer_name"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	var branchID uuid.UUID
	if auth.Role == "owner" {
		bID := c.Query("branch_id")
		if bID == "" {
			bID = payload.BranchID
		}
		if bID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "branch_id is required for owner (pass via query or body)"})
		}
		var err error
		branchID, err = uuid.Parse(bID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch_id format (must be UUID)"})
		}
	} else {
		if auth.BranchID == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "branch not assigned to your account"})
		}
		branchID = *auth.BranchID
	}

	input := CreateOrderInput{
		Type:         payload.Type,
		CustomerName: payload.CustomerName,
	}

	if payload.TableID != "" {
		tid, err := uuid.Parse(payload.TableID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid table_id format"})
		}
		input.TableID = &tid
	}

	o, err := h.svc.Create(auth, branchID, input)
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
//	@Failure		404	{object}	httputil.Error404Response
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
//	@Failure 400 {object} httputil.Error400Response
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
//	@Description	Add a menu item to an open order with optional modifier options. Validates modifiers belong to the menu item.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Order UUID"
//	@Param			body	body		AddOrderItemInput	true	"Order item payload"
//	@Success		201		{object}	OrderItem
//	@Failure 400 {object} httputil.Error400Response
//	@Failure 401 {object} httputil.Error401Response
//	@Failure 404 {object} httputil.Error404Response
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
//	@Failure 400 {object} httputil.Error400Response
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
//	@Failure 400 {object} httputil.Error400Response
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
