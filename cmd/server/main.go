package main

import (
	"fmt"
	"log"

	"github.com/bayarin/backend/config"
	"github.com/bayarin/backend/internal/auth"
	"github.com/bayarin/backend/internal/branch"
	"github.com/bayarin/backend/internal/business"
	"github.com/bayarin/backend/internal/dashboard"
	appdb "github.com/bayarin/backend/internal/db"
	"github.com/bayarin/backend/internal/menu"
	"github.com/bayarin/backend/internal/middleware"
	"github.com/bayarin/backend/internal/modifier"
	"github.com/bayarin/backend/internal/order"
	"github.com/bayarin/backend/internal/payment"
	"github.com/bayarin/backend/internal/staff"
	"github.com/bayarin/backend/internal/table"
	_ "github.com/bayarin/backend/docs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	swagger "github.com/gofiber/swagger"
)

//	@title			Bayar.in API
//	@version		1.0
//	@description	Multi-tenant POS SaaS backend
//	@host			localhost:8080
//	@BasePath		/api/v1
//	@securityDefinitions.apikey	BearerAuth
//	@in				header
//	@name			Authorization
//	@description	Paste your JWT token as: Bearer <token>
func main() {
	// Load environment configuration.
	config.Load()

	// Connect to database.
	db, err := appdb.Connect(config.App.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialise services.
	authSvc := auth.NewService(db)
	businessSvc := business.NewService(db)
	branchSvc := branch.NewService(db)
	staffSvc := staff.NewService(db)
	menuSvc := menu.NewService(db)
	modifierSvc := modifier.NewService(db)
	tableSvc := table.NewService(db)
	orderSvc := order.NewService(db)
	paymentSvc := payment.NewService(db)
	dashboardSvc := dashboard.NewService(db)

	// Initialise handlers.
	authHdlr := auth.NewHandler(authSvc)
	businessHdlr := business.NewHandler(businessSvc)
	branchHdlr := branch.NewHandler(branchSvc)
	staffHdlr := staff.NewHandler(staffSvc)
	menuHdlr := menu.NewHandler(menuSvc)
	modifierHdlr := modifier.NewHandler(modifierSvc)
	tableHdlr := table.NewHandler(tableSvc)
	orderHdlr := order.NewHandler(orderSvc)
	paymentHdlr := payment.NewHandler(paymentSvc)
	dashboardHdlr := dashboard.NewHandler(dashboardSvc)

	// Create Fiber app.
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false, "error": err.Error(),
			})
		},
	})

	// Global middleware.
	app.Use(recover.New())
	app.Use(logger.New())

	// Rate limiter for auth endpoints (10 req/min).
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 60,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false, "error": "too many requests",
			})
		},
	})

	// Convenience: auth middleware bound to this db.
	authMW := middleware.AuthMiddleware(db)

	// ── API v1 ──────────────────────────────────────────────────────────────
	api := app.Group("/api/v1")

	// ── AUTH (public) ──
	authRoute := api.Group("/auth")
	authRoute.Post("/register-owner", authLimiter, authHdlr.RegisterOwner)
	authRoute.Post("/login", authLimiter, authHdlr.Login)
	authRoute.Post("/logout", authMW, authHdlr.Logout)
	authRoute.Get("/me", authMW, authHdlr.Me)

	// ── BUSINESS [auth + owner] ──
	biz := api.Group("/business", authMW, middleware.RequireOwner())
	biz.Get("/", businessHdlr.Get)
	biz.Patch("/", businessHdlr.Update)

	// ── BRANCHES [auth + owner] ──
	branches := api.Group("/branches", authMW, middleware.RequireOwner())
	branches.Get("/", branchHdlr.List)
	branches.Post("/", branchHdlr.Create)
	branches.Patch("/:id", branchHdlr.Update)

	// ── STAFF [auth + owner] ──
	staffRoute := api.Group("/staff", authMW, middleware.RequireOwner())
	staffRoute.Get("/", staffHdlr.List)
	staffRoute.Post("/", staffHdlr.Create)
	staffRoute.Patch("/:id", staffHdlr.Update)

	// ── MENU [auth + owner] ──
	menuRoute := api.Group("/menu", authMW, middleware.RequireOwner())
	menuRoute.Get("/", menuHdlr.List)
	menuRoute.Post("/", menuHdlr.Create)
	menuRoute.Patch("/:id", menuHdlr.Update)

	// ── MODIFIERS [auth + owner] ──
	modifierRoute := api.Group("/modifiers", authMW, middleware.RequireOwner())
	modifierRoute.Get("/", modifierHdlr.List)
	modifierRoute.Post("/", modifierHdlr.Create)
	modifierRoute.Patch("/:id", modifierHdlr.Update)

	// ── TABLES [auth + owner] ──
	tableRoute := api.Group("/tables", authMW, middleware.RequireOwner())
	tableRoute.Get("/", tableHdlr.List)
	tableRoute.Post("/", tableHdlr.Create)
	tableRoute.Patch("/:id", tableHdlr.Update)

	// ── ORDERS [auth + cashier or owner] ──
	orderRoute := api.Group("/orders", authMW, middleware.RequireCashierOrOwner())
	orderRoute.Get("/", orderHdlr.List)
	orderRoute.Post("/", orderHdlr.Create)
	orderRoute.Get("/:id", orderHdlr.GetByID)
	orderRoute.Patch("/:id", orderHdlr.Update)

	// ── ORDER ITEMS ──
	orderRoute.Post("/:id/items", orderHdlr.AddItem)
	orderRoute.Patch("/:id/items/:item_id", orderHdlr.UpdateItem)
	orderRoute.Delete("/:id/items/:item_id", orderHdlr.DeleteItem)

	// ── PAYMENT [auth + cashier or owner] ──
	orderRoute.Post("/:id/pay", paymentHdlr.Pay)

	// ── DASHBOARD ──
	dash := api.Group("/dashboard", authMW)
	dash.Get("/owner", middleware.RequireOwner(), dashboardHdlr.OwnerDashboard)
	dash.Get("/cashier", middleware.RequireCashierOrOwner(), dashboardHdlr.CashierDashboard)

	// ── Swagger UI ──
	app.Get("/swagger/*", swagger.HandlerDefault)

	// ── Health check ──
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	addr := fmt.Sprintf(":%s", config.App.Port)
	log.Printf("🚀  Bayar.in API listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
