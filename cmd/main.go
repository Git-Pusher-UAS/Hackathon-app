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
	analyticshandler "backend-hackathon/internal/handlers/analytics"
	auditalerthandler "backend-hackathon/internal/handlers/auditalert"
	auditloghandler "backend-hackathon/internal/handlers/auditlog"
	bolsaextrahandler "backend-hackathon/internal/handlers/bolsaextra"
	carhandler "backend-hackathon/internal/handlers/caseaccessrequest"
	casehandler "backend-hackathon/internal/handlers/casefile"
	cataloghandler "backend-hackathon/internal/handlers/catalog"
	coursehandler "backend-hackathon/internal/handlers/course"
	employerhandler "backend-hackathon/internal/handlers/employer"
	helpeventhandler "backend-hackathon/internal/handlers/helpevent"
	ichandler "backend-hackathon/internal/handlers/institutionalcontact"
	jobhandler "backend-hackathon/internal/handlers/job"
	notehandler "backend-hackathon/internal/handlers/note"
	outreachhandler "backend-hackathon/internal/handlers/outreach"
	portalhandler "backend-hackathon/internal/handlers/portal"
	referralhandler "backend-hackathon/internal/handlers/referral"
	surveyhandler "backend-hackathon/internal/handlers/survey"
	userhandler "backend-hackathon/internal/handlers/user"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	analyticsmw "backend-hackathon/internal/middleware/analytics"
	auditlogmw "backend-hackathon/internal/middleware/auditlog"
	carmw "backend-hackathon/internal/middleware/caseaccessrequest"
	casemw "backend-hackathon/internal/middleware/casefile"
	coursemw "backend-hackathon/internal/middleware/course"
	outreachmw "backend-hackathon/internal/middleware/outreach"
	refmw "backend-hackathon/internal/middleware/referral"
	surveymw "backend-hackathon/internal/middleware/survey"
	usermiddleware "backend-hackathon/internal/middleware/user"
	gormrepo "backend-hackathon/internal/repository/gorm"
	"backend-hackathon/internal/security"
	analyticsservice "backend-hackathon/internal/services/analytics"
	auditalertservice "backend-hackathon/internal/services/auditalert"
	auditlogservice "backend-hackathon/internal/services/auditlog"
	carservice "backend-hackathon/internal/services/caseaccessrequest"
	caseservice "backend-hackathon/internal/services/casefile"
	catalogservice "backend-hackathon/internal/services/catalog"
	courseservice "backend-hackathon/internal/services/course"
	employerservice "backend-hackathon/internal/services/employer"
	helpeventservice "backend-hackathon/internal/services/helpevent"
	icservice "backend-hackathon/internal/services/institutionalcontact"
	jobservice "backend-hackathon/internal/services/job"
	noteservice "backend-hackathon/internal/services/note"
	outreachservice "backend-hackathon/internal/services/outreach"
	portalservice "backend-hackathon/internal/services/portal"
	referralservice "backend-hackathon/internal/services/referral"
	surveyservice "backend-hackathon/internal/services/survey"
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
	catalogHandler := cataloghandler.New(catalogservice.New(uow))
	icHandler := ichandler.New(icservice.New(uow))
	helpeventHandler := helpeventhandler.New(helpeventservice.New(uow))
	carHandler := carhandler.New(carservice.New(uow, keyring, cfg.JWT.Secret))
	referralHandler := referralhandler.New(referralservice.New(uow, keyring, cfg.JWT.Secret))
	noteHandler := notehandler.New(noteservice.New(caseService))
	auditLogHandler := auditloghandler.New(auditlogservice.New(uow))
	auditAlertHandler := auditalerthandler.New(auditalertservice.New(uow))
	analyticsHandler := analyticshandler.New(analyticsservice.New(uow))
	portalHandler := portalhandler.New(portalservice.New(uow, cfg.IsDev(), cfg.OTPSecret))
	courseHandler := coursehandler.New(courseservice.New(uow, cfg.JWT.Secret))
	outreachHandler := outreachhandler.New(outreachservice.New(uow))
	jobHandler := jobhandler.New(jobservice.New(uow))
	surveyHandler := surveyhandler.New(surveyservice.New(uow))
	employerHandler := employerhandler.New(employerservice.New(uow, cfg.IsDev(), cfg.OTPSecret))
	bolsaExtra := bolsaextrahandler.New(uow)
	app := fiber.New(fiber.Config{
		AppName:   handlers.ServiceName,
		BodyLimit: 256 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// /api usa el sobre de error del contrato; el resto conserva el
			// formato legado.
			if strings.HasPrefix(c.Path(), "/api/") {
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
	allowCredentials := cfg.Server.CORSOrigin != "*"
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.CORSOrigin,
		AllowCredentials: allowCredentials,
		AllowHeaders:     "Authorization, Content-Type, X-CSRF-Token, If-Match, X-Operation-Id",
		ExposeHeaders:    "ETag",
	}))

	statusHandler := handlers.NewStatusHandler(sqlDB, version)

	// Documentación Swagger
	app.Get("/docs/*", swagger.HandlerDefault)

	// Público
	app.Get("/health", handlers.Healthcheck)

	api := app.Group("/api")
	api.Get("/status", statusHandler.Status)
	api.Get("/institutional-contacts", icHandler.List)

	otpLimit := middleware.NewRateLimiter(10, time.Minute)
	strictOtpLimit := middleware.NewRateLimiter(20, 10*time.Minute)
	portal := api.Group("/portal")
	portal.Post("/auth/login", otpLimit.Middleware(func(c *fiber.Ctx) string { return "portal-login:" + c.IP() }), portalHandler.Login)
	portal.Post("/auth/verify", strictOtpLimit.Middleware(func(c *fiber.Ctx) string { return "portal-verify:" + c.IP() }), portalHandler.Verify)
	portal.Post("/accounts", otpLimit.Middleware(func(c *fiber.Ctx) string { return "portal-account:" + c.IP() }), portalHandler.CreateAccount)
	portalProtected := portal.Group("", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF())
	portalProtected.Get("/me", portalHandler.Me)
	portalProtected.Patch("/me", portalHandler.PatchMe)
	portalProtected.Post("/auth/logout", portalHandler.Logout)
	portalProtected.Get("/me/cv", bolsaExtra.GetCV)
	portalProtected.Put("/me/cv", bolsaExtra.PutCV)
	portalProtected.Get("/me/applications", bolsaExtra.MyApplications)
	portalProtected.Get("/me/applications/:id", bolsaExtra.MyApplication)
	portalProtected.Post("/me/applications/:id/interview-response", bolsaExtra.InterviewResponse)
	portalProtected.Post("/me/applications/:id/withdrawal", bolsaExtra.Withdraw)
	helpAlerts := api.Group("/help-alerts", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF())
	helpAlerts.Post("", portalHandler.CreateAlert)
	helpAlerts.Post("/:id/trusted-contacts", portalHandler.NotifyTrustedContacts)
	helpAlerts.Post("/:id/cancellation", portalHandler.CancelAlert)
	courseReadAuth := middleware.PortalOrStaffAuthMiddleware(cfg.JWT.Secret, uow, coursemw.PermManage)
	api.Get("/courses", courseReadAuth, usermiddleware.NoStore, courseHandler.List)
	api.Get("/courses/:id", courseReadAuth, usermiddleware.NoStore, courseHandler.Get)
	api.Post("/courses/:id/enrollments", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF(), courseHandler.Enroll)
	api.Get("/surveys/pending", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF(), surveyHandler.Pending)
	api.Get("/surveys/:id", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF(), surveyHandler.Get)
	api.Post("/surveys/:id/responses", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF(), surveyHandler.Submit)
	api.Get("/jobs", middleware.OptionalPortalAuthMiddleware(uow), jobHandler.List)
	api.Get("/jobs/:id", middleware.OptionalPortalAuthMiddleware(uow), jobHandler.Get)
	api.Post("/jobs/:id/applications", middleware.PortalAuthMiddleware(uow), middleware.PortalCSRF(), jobHandler.Apply)

	// Employers domain (aislado)
	emp := api.Group("/employers")
	emp.Post("/registrations", otpLimit.Middleware(func(c *fiber.Ctx) string { return "emp-reg:" + c.IP() }), employerHandler.Register)
	emp.Post("/auth/login", otpLimit.Middleware(func(c *fiber.Ctx) string { return "emp-login:" + c.IP() }), employerHandler.Login)
	emp.Post("/auth/verify", strictOtpLimit.Middleware(func(c *fiber.Ctx) string { return "emp-verify:" + c.IP() }), employerHandler.Verify)
	empProtected := emp.Group("/me", middleware.EmployerAuthMiddleware(uow), middleware.EmployerCSRF())
	empProtected.Get("/registration", employerHandler.MeRegistration)
	empProtected.Put("/registration", employerHandler.UpdateRegistration)
	employerJobs := api.Group("/employer", middleware.EmployerAuthMiddleware(uow), middleware.EmployerCSRF())
	employerJobs.Get("/jobs", employerHandler.ListMyJobs)
	employerJobs.Post("/jobs", employerHandler.CreateMyJob)
	employerJobs.Get("/jobs/:id", employerHandler.GetMyJob)
	employerJobs.Patch("/jobs/:id", employerHandler.PatchMyJob)
	employerJobs.Get("/jobs/:id/applications", employerHandler.ListJobApplications)
	employerJobs.Get("/applications/:id", employerHandler.GetApplication)
	employerJobs.Post("/applications/:id/interview", employerHandler.Interview)
	employerJobs.Post("/applications/:id/decision", employerHandler.Decide)

	authMw := middleware.AuthMiddleware(cfg.JWT.Secret, uow)

	// Igual que authMw, pero si no hay credenciales de sesión deja pasar la petición.
	optionalAuth := func(c *fiber.Ctx) error {
		if c.Get(fiber.HeaderAuthorization) == "" && c.Cookies("sai_session") == "" {
			return c.Next()
		}
		return authMw(c)
	}

	// Público / arranque de sesión
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/activations", authHandler.Activate)
	api.Post("/auth/mfa/verify", optionalAuth, authHandler.VerifyMFA)
	api.Get("/auth/csrf-token", optionalAuth, authHandler.CSRFToken)

	// Protegido: sesión por cookie o Bearer + CSRF en escrituras
	protected := api.Group("", authMw, middleware.CSRF(uow))
	protected.Get("/auth/me", authHandler.Me)
	protected.Post("/auth/mfa/enrollments", authHandler.EnrollMFA)
	protected.Post("/auth/logout", authHandler.Logout)

	// Contrato de la API. Comparte el mismo JWT y añade CSRF en escrituras.
	apiContract := app.Group("/api", middleware.AuthMiddleware(cfg.JWT.Secret, uow), usermiddleware.NoStore)
	apiContract.Get("/catalogs", catalogHandler.Catalogs)
	apiContract.Post("/help-events", usermiddleware.RequireCSRF, helpeventHandler.Create)
	apiContract.Get("/users/me", userHandler.Me)
	apiContract.Get("/users/me/sessions", userHandler.ListMySessions)
	apiContract.Post("/users/me/sessions/revoke-others", usermiddleware.RequireCSRF, userHandler.RevokeOthers)
	apiContract.Delete("/users/me/sessions/:sessionId", usermiddleware.RequireCSRF, userHandler.RevokeMySession)

	staff := apiContract.Group("/users/staff", middleware.RequirePermiso("admin"))
	staff.Get("", userHandler.List)
	staff.Post("", usermiddleware.RequireCSRF, userHandler.Create)
	staff.Get("/:userId", userHandler.Get)
	staff.Patch("/:userId", usermiddleware.RequireCSRF, userHandler.Update)
	staff.Post("/:userId/force-logout", usermiddleware.RequireCSRF, userHandler.ForceLogout)
	staff.Post("/:userId/session-revocations", usermiddleware.RequireCSRF, userHandler.RevokeAllSessions)
	staff.Post("/:userId/mfa-resets", usermiddleware.RequireCSRF, userHandler.ResetMFA)
	// Compatibilidad alias /api/users* (evita sombrear /users/me*)
	usersAlias := apiContract.Group("/users", middleware.RequirePermiso("admin"), func(c *fiber.Ctx) error {
		if uid := c.Params("userId"); uid == "me" {
			return c.Next()
		}
		return c.Next()
	})
	usersAlias.Get("", userHandler.List)
	usersAlias.Post("", usermiddleware.RequireCSRF, userHandler.Create)
	// rutas paramétricas: el guard anterior no puede saltar a otra ruta; se mantienen en /users/staff/*
	// para evitar shadowing de /users/me. Se deja solo alias de colección.
	csrf := usermiddleware.RequireCSRF
	apiContract.Post("/case-access-requests", csrf, carmw.Require(carmw.PermRequest), carHandler.Create)
	apiContract.Get("/case-access-requests", carmw.Require(carmw.PermReview), carHandler.List)
	apiContract.Post("/case-access-requests/:id/rejection", csrf, carmw.Require(carmw.PermReview), carHandler.Reject)
	apiContract.Get("/referrals", refmw.Require(refmw.PermRead), referralHandler.List)
	apiContract.Post("/referrals/:referralId/transitions", csrf, refmw.Require(refmw.PermTransition), referralHandler.Transition)
	apiContract.Post("/notes/sync", csrf, casemw.Require(casemw.PermWrite), noteHandler.Sync)
	apiContract.Get("/audit-logs", auditlogmw.Require(), auditLogHandler.List)
	apiContract.Get("/audit-alerts", auditlogmw.Require(), auditAlertHandler.List)
	apiContract.Get("/analytics/summary", analyticsmw.Require(analyticsmw.PermRead), analyticsHandler.Summary)
	apiContract.Get("/analytics/heatmap", analyticsmw.Require(analyticsmw.PermHeatmap), analyticsHandler.Heatmap)
	apiContract.Get("/surveys/results", surveymw.Require(surveymw.PermResults), surveyHandler.Results)
	apiContract.Post("/jobs", csrf, coursemw.Require(coursemw.PermManage), jobHandler.Create)
	admin := apiContract.Group("/admin", coursemw.Require(coursemw.PermManage))
	admin.Get("/jobs", bolsaExtra.AdminListJobs)
	admin.Post("/jobs", csrf, bolsaExtra.AdminCreateJob)
	admin.Get("/jobs/:id", bolsaExtra.AdminGetJob)
	admin.Post("/jobs/:id/review", csrf, bolsaExtra.AdminReviewJob)
	admin.Get("/employers", bolsaExtra.AdminListEmployers)
	admin.Post("/employers/:id/review", csrf, bolsaExtra.AdminReviewEmployer)
	admin.Get("/hires", bolsaExtra.AdminHires)
	admin.Post("/applications/:id/contact-release", csrf, bolsaExtra.AdminContactRelease)

	courses := apiContract.Group("/courses", coursemw.Require(coursemw.PermManage))
	courses.Post("", csrf, courseHandler.Create)
	courses.Get("/:id/attendance", courseHandler.Attendance)
	courses.Put("/:id/attendance", csrf, courseHandler.SetAttendance)

	outreach := apiContract.Group("/outreach-visits", outreachmw.Require(outreachmw.PermManage))
	outreach.Get("", outreachHandler.List)
	outreach.Post("", csrf, outreachHandler.Create)

	cf := apiContract.Group("/case-files")
	cf.Post("/duplicate-checks", csrf, casemw.Require(casemw.PermWrite), caseHandler.DuplicateCheck)
	cf.Post("", csrf, casemw.Require(casemw.PermWrite), caseHandler.Create)
	cf.Get("", casemw.Require(casemw.PermRead), caseHandler.List)
	cf.Get("/:recordId", casemw.Require(casemw.PermRead), caseHandler.Get)
	cf.Patch("/:recordId", csrf, casemw.Require(casemw.PermWrite), caseHandler.Update)
	cf.Get("/:recordId/timeline", casemw.Require(casemw.PermRead), caseHandler.Timeline)
	cf.Post("/:recordId/disclosures", csrf, casemw.Require(casemw.PermDisclose), caseHandler.Disclose)
	cf.Post("/:recordId/assignments", csrf, casemw.Require(casemw.PermAssign), caseHandler.Assign)
	cf.Post("/:recordId/assignments/:assignmentId/revocations", csrf, casemw.Require(casemw.PermAssign), caseHandler.RevokeAssignment)
	cf.Get("/:recordId/departments/:departmentId/notes", casemw.Require(casemw.PermRead), caseHandler.ListNotes)
	cf.Post("/:recordId/departments/:departmentId/notes", csrf, casemw.Require(casemw.PermWrite), caseHandler.CreateNote)
	cf.Get("/:recordId/consents", casemw.Require(casemw.PermRead), caseHandler.ListConsents)
	cf.Post("/:recordId/consents", csrf, casemw.Require(casemw.PermConsent), caseHandler.RecordConsent)
	cf.Post("/:recordId/consents/:consentId/revocations", csrf, casemw.Require(casemw.PermConsent), caseHandler.RevokeConsent)
	cf.Post("/:recordId/referrals", csrf, casemw.Require(casemw.PermRefer), caseHandler.CreateReferral)

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
