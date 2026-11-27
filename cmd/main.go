package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "backend-hackathon/docs"
	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"
	casehandler "backend-hackathon/internal/handlers/casefile"
	userhandler "backend-hackathon/internal/handlers/user"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	casemw "backend-hackathon/internal/middleware/casefile"
	usermiddleware "backend-hackathon/internal/middleware/user"
	gormrepo "backend-hackathon/internal/repository/gorm"
	"backend-hackathon/internal/security"
	caseservice "backend-hackathon/internal/services/casefile"
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
	if cfg.CURPKey == "" {
		log.Println("warning: CURP_HMAC_KEY no configurada; POST /case-files y duplicate-checks responder\u00e1n 503")
	}
	caseService := caseservice.New(uow, keyring, cfg.JWT.Secret, cfg.CURPKey)
	caseHandler := casehandler.New(caseService)
	app := fiber.New(fiber.Config{
		AppName:   handlers.ServiceName,
		BodyLimit: 256 * 1024,
		// Toda la API responde errores con el sobre del contrato:
		// {"error": {"code", "message", "requestId", "fields"?}}.
		ErrorHandler: httpx.Write,
	})

	// Orden: recover -> requestid -> logger -> cors.
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path} (${ip}) rid=${locals:requestid}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:  cfg.Server.CORSOrigin,
		AllowHeaders:  "Authorization, Content-Type, X-CSRF-Token, If-Match, X-Operation-Id",
		ExposeHeaders: "ETag",
	}))

	statusHandler := handlers.NewStatusHandler(sqlDB, version)

	// Documentación Swagger
	app.Get("/docs/*", swagger.HandlerDefault)

	// Público
	app.Get("/health", handlers.Healthcheck)

	api := app.Group("/api")
	api.Get("/status", statusHandler.Status)

	// Base del contrato: /api/v1.
	v1 := app.Group("/api/v1")
	v1.Get("/health", handlers.Healthcheck)

	// Acceso del personal, público y sin CSRF. /api/auth/* queda como alias
	// temporal de /api/v1/auth/*. Se registran antes de los grupos protegidos,
	// que se montan como middleware sobre todo su prefijo.
	for _, g := range []fiber.Router{v1, api} {
		g.Post("/auth/login", authHandler.Login)
		g.Post("/auth/mfa/enrollments", authHandler.EnrollMFA)
		g.Post("/auth/mfa/verify", authHandler.VerifyMFA)
	}

	// Protegido: JWT (cookie sai_staff o Bearer) + sesión revocable, sin caché
	// y X-CSRF-Token en toda escritura.
	authMw := middleware.AuthMiddleware(cfg.JWT.Secret, uow)
	csrfWrites := func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		return usermiddleware.RequireCSRF(c)
	}

	legacy := api.Group("/auth", authMw, usermiddleware.NoStore, csrfWrites)
	legacy.Get("/me", authHandler.Me)
	legacy.Get("/csrf-token", authHandler.CSRFToken)
	legacy.Post("/logout", authHandler.Logout)

	priv := v1.Group("", authMw, usermiddleware.NoStore, csrfWrites)
	priv.Get("/auth/me", authHandler.Me)
	priv.Get("/auth/csrf-token", authHandler.CSRFToken)
	priv.Post("/auth/logout", authHandler.Logout)

	// /users/me va antes del grupo admin, que exige el permiso "admin" para todo /users.
	priv.Get("/users/me", userHandler.Me)
	priv.Get("/users/me/sessions", userHandler.ListMySessions)
	priv.Delete("/users/me/sessions/:sessionId", userHandler.RevokeMySession)

	admin := priv.Group("/users", middleware.RequirePermiso("admin"))
	admin.Get("", userHandler.List)
	admin.Post("", userHandler.Create)
	admin.Patch("/:userId", userHandler.Update)
	admin.Post("/:userId/session-revocations", userHandler.RevokeAllSessions)
	admin.Post("/:userId/mfa-resets", userHandler.ResetMFA)

	cf := priv.Group("/case-files")
	cf.Post("/duplicate-checks", casemw.Require(casemw.PermWrite), caseHandler.DuplicateCheck)
	cf.Post("", casemw.Require(casemw.PermWrite), caseHandler.Create)
	cf.Get("", casemw.Require(casemw.PermRead), caseHandler.List)
	cf.Get("/:recordId", casemw.Require(casemw.PermRead), caseHandler.Get)
	cf.Patch("/:recordId", casemw.Require(casemw.PermWrite), caseHandler.Update)
	cf.Get("/:recordId/timeline", casemw.Require(casemw.PermRead), caseHandler.Timeline)
	cf.Post("/:recordId/disclosures", casemw.Require(casemw.PermDisclose), caseHandler.Disclose)
	cf.Post("/:recordId/assignments", casemw.Require(casemw.PermAssign), caseHandler.Assign)
	cf.Post("/:recordId/assignments/:assignmentId/revocations", casemw.Require(casemw.PermAssign), caseHandler.RevokeAssignment)
	cf.Get("/:recordId/departments/:departmentId/notes", casemw.Require(casemw.PermRead), caseHandler.ListNotes)
	cf.Post("/:recordId/departments/:departmentId/notes", casemw.Require(casemw.PermWrite), caseHandler.CreateNote)
	cf.Get("/:recordId/consents", casemw.Require(casemw.PermRead), caseHandler.ListConsents)
	cf.Post("/:recordId/consents", casemw.Require(casemw.PermConsent), caseHandler.RecordConsent)
	cf.Post("/:recordId/consents/:consentId/revocations", casemw.Require(casemw.PermConsent), caseHandler.RevokeConsent)
	cf.Post("/:recordId/referrals", casemw.Require(casemw.PermRefer), caseHandler.CreateReferral)

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
