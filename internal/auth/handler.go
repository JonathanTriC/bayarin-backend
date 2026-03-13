package auth

import (
	"strings"

	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP for the auth module.
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterOwner godoc
//
//	@Summary		Register a new business owner
//	@Description	Creates a new business and an owner account atomically
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterOwnerInput				true	"Registration payload"
//	@Success		201		{object}	UserResponse
//	@Failure		400		{object}	httputil.ErrorResponse
//	@Router			/auth/register-owner [post]
func (h *Handler) RegisterOwner(c *fiber.Ctx) error {
	var input RegisterOwnerInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "invalid request body",
		})
	}

	user, err := h.svc.RegisterOwner(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true, "data": user,
	})
}

// Login godoc
//
//	@Summary		Login
//	@Description	Authenticate user and return a JWT token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginInput				true	"Login payload"
//	@Success		200		{object}	LoginResponse
//	@Failure		401		{object}	httputil.ErrorResponse
//	@Router			/auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "invalid request body",
		})
	}

	resp, err := h.svc.Login(input)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false, "error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": resp})
}

// Logout godoc
//
//	@Summary		Logout
//	@Description	Revoke the current session token
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	httputil.MessageResponse
//	@Failure		500	{object}	httputil.ErrorResponse
//	@Router			/auth/logout [post]
func (h *Handler) Logout(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if err := h.svc.Logout(token); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "could not revoke session",
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": "logged out"})
}

// Me godoc
//
//	@Summary		Get current user
//	@Description	Returns the authenticated user's profile
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserResponse
//	@Failure		404	{object}	httputil.ErrorResponse
//	@Router			/auth/me [get]
func (h *Handler) Me(c *fiber.Ctx) error {
	auth := c.Locals("auth").(middleware.AuthContext)
	user, err := h.svc.Me(auth.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false, "error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": user})
}
