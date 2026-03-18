package qris

import (
	"github.com/bayarin/backend/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// GenerateQRISInput represents input for generate dynamic QRIS
type GenerateQRISInput struct {
	BranchID string `json:"branch_id,omitempty"` // only required for owner role
	Amount   int64  `json:"amount"`
}

// @Summary      Upload static QRIS
// @Description  Upload a static QRIS image for a branch. Decodes the QR, validates CRC-16, stores image in Supabase Storage, saves QRIS string to DB. Deactivates previous QRIS for the branch.
// @Tags         qris
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        image      formData  file    true  "QRIS image file (PNG or JPG, max 2MB)"
// @Param        branch_id  formData  string  true  "Branch UUID"
// @Success      201  {object}  map[string]interface{}  "QRIS uploaded and stored successfully"
// @Failure      400  {object}  map[string]interface{}  "Invalid image, invalid QRIS string, or CRC mismatch"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden - owner only"
// @Router       /qris/upload [post]
func (h *Handler) Upload(c *fiber.Ctx) error {
	branchIDStr := c.FormValue("branch_id")
	branchID, err := uuid.Parse(branchIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid branch_id format",
		})
	}

	authCtx, ok := c.Locals("auth").(middleware.AuthContext)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "unauthorized",
		})
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "image file is required",
		})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "failed to read image file",
		})
	}
	defer file.Close()

	record, err := h.svc.UploadQRIS(c.Context(), branchID, authCtx.UserID, file, fileHeader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    record,
	})
}

// @Summary      Generate dynamic QRIS
// @Description  Retrieves the active static QRIS for a branch, injects the transaction amount, recalculates CRC-16, and returns the dynamic QRIS string and QR image (base64 PNG). Must complete in < 200ms.
// @Tags         qris
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  GenerateQRISInput  true  "Branch ID and transaction amount"
// @Success      200   {object}  map[string]interface{}  "Dynamic QRIS string and image"
// @Failure      400   {object}  map[string]interface{}  "Missing amount or no active QRIS"
// @Failure      401   {object}  map[string]interface{}  "Unauthorized"
// @Router       /qris/generate [post]
func (h *Handler) Generate(c *fiber.Ctx) error {
	var input GenerateQRISInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid JSON body",
		})
	}

	auth := c.Locals("auth").(middleware.AuthContext)
	
	var branchID uuid.UUID
	if auth.BranchID != nil {
		branchID = *auth.BranchID
	} else {
		if input.BranchID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "branch_id is required",
			})
		}
		var errParse error
		branchID, errParse = uuid.Parse(input.BranchID)
		if errParse != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "invalid branch_id format",
			})
		}
	}

	if input.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "amount must be greater than 0",
		})
	}

	dynQRIS, b64Image, amount, err := h.svc.GenerateQRIS(c.Context(), branchID, input.Amount)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"qris_string":       dynQRIS,
			"qris_image_base64": b64Image,
			"amount":            amount,
		},
	})
}

// @Summary      QRIS upload history
// @Description  Returns all QRIS records for a branch. Image URLs are signed Supabase Storage URLs (1 hour expiry).
// @Tags         qris
// @Produce      json
// @Security     BearerAuth
// @Param        branch_id  query  string  true  "Branch UUID"
// @Success      200  {object}  map[string]interface{}  "QRIS history"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden - owner only"
// @Router       /qris/history [get]
func (h *Handler) History(c *fiber.Ctx) error {
	branchIDStr := c.Query("branch_id")
	branchID, err := uuid.Parse(branchIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid branch_id format",
		})
	}

	history, err := h.svc.GetHistory(c.Context(), branchID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    history,
	})
}
