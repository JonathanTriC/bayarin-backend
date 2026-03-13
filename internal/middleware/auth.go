package middleware

import (
	"database/sql"
	"strings"
	"time"

	"github.com/bayarin/backend/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthContext carries the authenticated user info injected into every request.
type AuthContext struct {
	UserID     uuid.UUID
	Role       string
	BusinessID uuid.UUID
	BranchID   *uuid.UUID
}

// AuthMiddleware validates the Bearer JWT, checks the session table, then
// injects AuthContext into c.Locals("auth").
func AuthMiddleware(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false, "error": "missing or invalid authorization header",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse & validate JWT signature + expiry.
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(config.App.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false, "error": "invalid token",
			})
		}

		// Verify session exists in DB and is not revoked / expired.
		var (
			userID     string
			businessID string
			branchID   sql.NullString
			role       string
			revoked    bool
			expiresAt  time.Time
		)

		row := db.QueryRowContext(c.Context(),
			`SELECT u.id, u.business_id, u.branch_id, u.role,
			        s.revoked, s.expires_at
			 FROM sessions s
			 JOIN users u ON u.id = s.user_id
			 WHERE s.token = $1`, tokenStr)

		if err := row.Scan(&userID, &businessID, &branchID, &role, &revoked, &expiresAt); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false, "error": "session not found",
			})
		}

		if revoked {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false, "error": "session revoked",
			})
		}
		if time.Now().After(expiresAt) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false, "error": "session expired",
			})
		}

		uid, _ := uuid.Parse(userID)
		bid, _ := uuid.Parse(businessID)

		authCtx := AuthContext{
			UserID:     uid,
			Role:       role,
			BusinessID: bid,
		}
		if branchID.Valid {
			parsed, _ := uuid.Parse(branchID.String)
			authCtx.BranchID = &parsed
		}

		c.Locals("auth", authCtx)
		return c.Next()
	}
}

// RequireOwner rejects non-owner users.
func RequireOwner() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals("auth").(AuthContext)
		if !ok || auth.Role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false, "error": "owner access required",
			})
		}
		return c.Next()
	}
}

// RequireCashierOrOwner allows both roles.
func RequireCashierOrOwner() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals("auth").(AuthContext)
		if !ok || (auth.Role != "owner" && auth.Role != "cashier") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false, "error": "cashier or owner access required",
			})
		}
		return c.Next()
	}
}
