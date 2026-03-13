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
//	@Description	Returns all tables. Filter by branch_id query param (optional).
//	@Tags			Tables
//	@Produce		json
//	@Security		BearerAuth
//	@Param			branch_id	query		string	false	"Branch UUID to filter tables"
//	@Success		200			{array}		Table
//	@Failure		500			{object}	httputil.ErrorResponse
//	@Router			/tables [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var branchIDFilter *uuid.UUID
	if bid := c.Query("branch_id"); bid != "" {
		parsed, err := uuid.Parse(bid)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch_id"})
		}
		branchIDFilter = &parsed
	}
	tables, err := h.svc.List(auth.BusinessID, branchIDFilter)
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
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/tables [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateTableInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	t, err := h.svc.Create(auth.BusinessID, input)
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
//	@Failure		400		{object}	httputil.ErrorResponse
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
