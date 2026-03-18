package table

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the table module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new table handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List tables
//	@Description	Returns all tables. Filter by branch_id query param (optional). Accessible by owner and cashier.
//	@Tags			Tables
//	@Produce		json
//	@Security		BearerAuth
//	@Param			branch_id	query		string	false	"Branch UUID to filter tables"
//	@Success		200			{array}		Table
//	@Failure		401			{object}	httputil.Error401Response
//	@Failure		500			{object}	httputil.Error500Response
//	@Router			/tables [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var branchIDFilter *uuid.UUID
	if auth.Role == "owner" {
		if bid := c.Query("branch_id"); bid != "" {
			parsed, err := uuid.Parse(bid)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch_id"})
			}
			branchIDFilter = &parsed
		}
	} else {
		// cashier - scoped automatically
		if auth.BranchID == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "branch not assigned"})
		}
		branchIDFilter = auth.BranchID
	}
	tables, err := h.svc.List(auth.BusinessID, branchIDFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": tables})
}

// Search godoc
//
//	@Summary		Search tables
//	@Description	Search tables by name (case-insensitive). Optionally filter by branch_id and/or status. Accessible by owner and cashier.
//	@Tags			Tables
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q			query		string	false	"Search keyword (matches table name)"
//	@Param			branch_id	query		string	false	"Branch UUID to filter tables"
//	@Param			status		query		string	false	"Filter by status: available or occupied"
//	@Success		200			{array}		Table
//	@Failure		401			{object}	httputil.Error401Response
//	@Failure		500			{object}	httputil.Error500Response
//	@Router			/tables/search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var branchIDFilter *uuid.UUID
	if auth.Role == "owner" {
		if bid := c.Query("branch_id"); bid != "" {
			parsed, err := uuid.Parse(bid)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch_id"})
			}
			branchIDFilter = &parsed
		}
	} else {
		if auth.BranchID == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "branch not assigned"})
		}
		branchIDFilter = auth.BranchID
	}
	tables, err := h.svc.Search(auth.BusinessID, c.Query("q"), branchIDFilter, c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": tables})
}

// Create godoc
//
//	@Summary		Create table
//	@Description	Create a new table under a branch. The branch must belong to the authenticated owner's business.
//	@Tags			Tables
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateTableInput	true	"Table payload"
//	@Success		201		{object}	Table
//	@Failure 400 {object} httputil.Error400Response
//	@Router			/tables [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	
	// CreateTable requires branch_id but we just parse it from body uniquely for Owner manually!
	// Wait, the prompt states: Create table is owner-only, so branch_id stays in request body!
	// However I already removed it from CreateTableInput structure to keep service clean!
	// So let's parse branch_id explicitly natively!
	var payload struct {
		BranchID uuid.UUID `json:"branch_id"`
		Name     string    `json:"name"`
		QRCode   string    `json:"qr_code"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	var branchID uuid.UUID
	if auth.Role == "owner" {
		branchID = payload.BranchID
	} else {
		if auth.BranchID == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "branch not assigned to your account"})
		}
		branchID = *auth.BranchID
	}

	t, err := h.svc.Create(auth.BusinessID, branchID, CreateTableInput{
		Name:   payload.Name,
		QRCode: payload.QRCode,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": t})
}

// Update godoc
//
//	@Summary		Update table
//	@Description	Partially update a table's name, status ("available"|"occupied"), or QR code
//	@Tags			Tables
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Table UUID"
//	@Param			body	body		UpdateTableInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	Table
//	@Failure 400 {object} httputil.Error400Response
//	@Router			/tables/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	tableID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid table id"})
	}
	var input UpdateTableInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	t, err := h.svc.Update(auth.BusinessID, tableID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": t})
}

// Reserve godoc
//
//	@Summary		Reserve a table
//	@Description	Mark a table as reserved. Cannot reserve an occupied table.
//	@Tags			tables
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string				true	"Table UUID"
//	@Param			body	body	ReserveTableInput	true	"Reservation note"
//	@Success		200		{object}	map[string]interface{}	"Table reserved"
//	@Failure		400		{object}	map[string]interface{}	"Table is occupied"
//	@Failure		401		{object}	map[string]interface{}	"Unauthorized"
//	@Router			/tables/{id}/reserve [patch]
func (h *Handler) Reserve(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	tableID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid table id"})
	}
	if auth.BranchID == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "error": "only cashiers assigned to a branch can reserve tables natively"})
	}

	var input ReserveTableInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	t, err := h.svc.Reserve(auth.BusinessID, tableID, *auth.BranchID, auth.UserID, input.Note)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": t})
}

// ClearStatus godoc
//
//	@Summary		Clear table status
//	@Description	Manually set table back to available. Cannot clear if table has an active open order.
//	@Tags			tables
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Table UUID"
//	@Success		200	{object}	map[string]interface{}	"Table cleared"
//	@Failure		400	{object}	map[string]interface{}	"Active order exists on this table"
//	@Failure		401	{object}	map[string]interface{}	"Unauthorized"
//	@Router			/tables/{id}/clear [patch]
func (h *Handler) ClearStatus(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	tableID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid table id"})
	}
	if auth.BranchID == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "error": "only cashiers assigned to a branch can clear tables natively"})
	}

	t, err := h.svc.ClearStatus(auth.BusinessID, tableID, *auth.BranchID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": t})
}
