package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "backend-hackathon/docs"
	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"
	userhandler "backend-hackathon/internal/handlers/user"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	usermiddleware "backend-hackathon/internal/middleware/user"
	gormrepo "backend-hackathon/internal/repository/gorm"
	"backend-hackathon/internal/security"
	userservice "backend-hackathon/internal/services/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
)

// version se puede sobrescribir en el build:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd
var version = "dev"

// @title           SAI API
// @version         1.0
// @description     Documentación de endpoints de la API.
// @host            localhost:3000
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := gormrepo.Open(&cfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// El UnitOfWork es el punto de entrada para la capa de servicios.
	uow := gormrepo.NewUnitOfWork(db)

	// La clave de cifrado de semillas TOTP debe ser base64 de 32 bytes.
	mfaKey, err := base64.RawStdEncoding.DecodeString(cfg.MFA.Secret)
	if err != nil {
		mfaKey, err = base64.StdEncoding.DecodeString(cfg.MFA.Secret)
	}
	if err != nil {
		log.Fatalf("mfa key: %v", err)
	}

	authService, err := auth.NewService(uow, cfg.JWT.Secret, cfg.JWT.ExpiresIn, mfaKey)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	authHandler := handlers.NewAuthHandler(authService)

	// La KEK solo es obligatoria para las operaciones cifradas (API-013/014).
	// Sin ella la API arranca, pero esos endpoints responden 503.
	var keyring *security.Keyring
	if len(cfg.KEK.Master) > 0 {
		keyring, err = security.NewKeyring(cfg.KEK.Master, cfg.KEK.Version)
		if err != nil {
			log.Fatalf("kek: %v", err)
		}
	} else {
		log.Println("warning: KEK_MASTER no configurada; API-013/API-014 responderán 503")
	}

	userService := userservice.New(uow, keyring, cfg.JWT.Secret)
	userHandler := userhandler.New(userService)

	app := fiber.New(fiber.Config{
		AppName:   handlers.ServiceName,
		BodyLimit: 256 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// /api/v1 usa el sobre de error del contrato; /api conserva el
			// formato legado.
			if strings.HasPrefix(c.Path(), "/api/v1/") {
				return httpx.Write(c, err)
			}
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Orden: recover -> requestid -> logger -> cors.
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path} (${ip}) rid=${locals:requestid}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:  cfg.Server.CORSOrigin,
		AllowHeaders:  "Authorization, Content-Type, X-CSRF-Token, If-Match",
		ExposeHeaders: "ETag",
	}))

	statusHandler := handlers.NewStatusHandler(sqlDB, version)

	// Documentación Swagger
	app.Get("/docs/*", swagger.HandlerDefault)

	// Público
	app.Get("/health", handlers.Healthcheck)

	api := app.Group("/api")
	api.Get("/status", statusHandler.Status)

	authMw := middleware.AuthMiddleware(cfg.JWT.Secret, uow)

	// Igual que authMw, pero si no hay header Authorization deja pasar la petición.
	optionalAuth := func(c *fiber.Ctx) error {
		if c.Get(fiber.HeaderAuthorization) == "" {
			return c.Next()
		}
		return authMw(c)
	}

	// Público
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/mfa/verify", optionalAuth, authHandler.VerifyMFA)

	// Protegido: JWT + sesión revocable
	protected := api.Group("", authMw, middleware.CSRF(uow))
	protected.Get("/auth/me", authHandler.Me)
	protected.Get("/auth/csrf-token", authHandler.CSRFToken)
	protected.Post("/auth/mfa/enrollments", authHandler.EnrollMFA)
	protected.Post("/auth/logout", authHandler.Logout)

	// /api/v1: contrato nuevo. Comparte el mismo JWT y añade CSRF en escrituras.
	v1 := app.Group("/api/v1", middleware.AuthMiddleware(cfg.JWT.Secret, uow), usermiddleware.NoStore)
	v1.Get("/users/me", userHandler.Me)
	v1.Get("/users/me/sessions", userHandler.ListMySessions)
	v1.Delete("/users/me/sessions/:sessionId", usermiddleware.RequireCSRF, userHandler.RevokeMySession)

	admin := v1.Group("/users", middleware.RequirePermiso("admin"))
	admin.Get("", userHandler.List)
	admin.Post("", usermiddleware.RequireCSRF, userHandler.Create)
	admin.Patch("/:userId", usermiddleware.RequireCSRF, userHandler.Update)
	admin.Post("/:userId/session-revocations", usermiddleware.RequireCSRF, userHandler.RevokeAllSessions)
	admin.Post("/:userId/mfa-resets", usermiddleware.RequireCSRF, userHandler.ResetMFA)

	// Apagado ordenado: cierra Fiber y luego el pool de GORM (defer).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(":" + cfg.Server.Port)
	}()

	log.Printf("API escuchando en :%s (version %s)", cfg.Server.Port, version)

	select {
	case err := <-errCh:
		log.Fatalf("server: %v", err)
	case <-ctx.Done():
		log.Println("apagando servidor...")
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}

	log.Println("servidor detenido")
}
