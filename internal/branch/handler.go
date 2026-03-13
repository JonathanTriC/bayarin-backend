package branch

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the HTTP layer for the branch module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new branch handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List godoc
//
//	@Summary		List branches
//	@Description	Returns all branches that belong to the authenticated owner's business
//	@Tags			Branches
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		Branch
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/branches [get]
func (h *Handler) List(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	branches, err := h.svc.List(auth.BusinessID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": branches})
}

// Create godoc
//
//	@Summary		Create branch
//	@Description	Create a new branch under the authenticated owner's business
//	@Tags			Branches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateBranchInput	true	"Branch payload"
//	@Success		201		{object}	Branch
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/branches [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	var input CreateBranchInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	b, err := h.svc.Create(auth.BusinessID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": b})
}

// Update godoc
//
//	@Summary		Update branch
//	@Description	Partially update a branch's name, address, or active status
//	@Tags			Branches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Branch UUID"
//	@Param			body	body		UpdateBranchInput	true	"Update payload (all fields optional)"
//	@Success		200		{object}	Branch
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/branches/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	branchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid branch id"})
	}
	var input UpdateBranchInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	b, err := h.svc.Update(auth.BusinessID, branchID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": b})
}
