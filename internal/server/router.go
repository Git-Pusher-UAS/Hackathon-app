package server

import (
	"errors"
	"html"
	"net/url"
	"path"
	"strings"
	"time"

	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

// router registra las rutas y encadena los middlewares.
//
// Orden (de fuera hacia dentro): Recover → RequestID → Logging → CORS → rutas.
func router(cfg *config.Config, db handlers.Pinger, version string) *fiber.App {
	status := handlers.NewStatus(db, version)
	app := fiber.NewWithCustomCtx(func(app *fiber.App) fiber.CustomCtx {
		return &routingContext{DefaultCtx: fiber.NewDefaultCtx(app)}
	}, fiber.Config{
		ReadBufferSize:               (1 << 20) + 4096,
		StreamRequestBody:            true,
		RequestMethods:               append(append([]string(nil), fiber.DefaultMethods...), fallbackMethod),
		DisableFlashMessages:         true,
		CaseSensitive:                true,
		StrictRouting:                true,
		ReadTimeout:                  15 * time.Second,
		WriteTimeout:                 15 * time.Second,
		IdleTimeout:                  60 * time.Second,
		DisableDefaultContentType:    true,
		DisablePreParseMultipartForm: true,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Los errores de transporte (400, 413, 431...) conservan su
			// código; no deben convertirse en fallos internos de la API.
			var protocolError *fiber.Error
			if errors.As(err, &protocolError) && protocolError.Code != fiber.StatusInternalServerError {
				return fiber.DefaultErrorHandler(c, err)
			}
			return httpx.Error(c, fiber.StatusInternalServerError, "internal_error", "error interno del servidor")
		},
	})
	app.Use(middleware.Recover(), middleware.RequestID, middleware.Logging(), middleware.CORS(cfg.Server.CORSOrigin))
	app.Use(canonicalPath)
	app.Get("/health", status.Health)
	app.Get("/api/v1/status", status.Status)

	notAllowed := func(c fiber.Ctx) error {
		c.Set(fiber.HeaderAllow, "GET, OPTIONS")
		return httpx.Error(c, fiber.StatusMethodNotAllowed, "method_not_allowed", "método no permitido")
	}
	app.All("/health", notAllowed)
	app.All("/api/v1/status", notAllowed)
	app.Use(notFound)
	return app
}

func notFound(c fiber.Ctx) error {
	return httpx.Error(c, fiber.StatusNotFound, "not_found", "ruta no encontrada")
}

// canonicalPath conserva la limpieza con redirección de ServeMux. Fiber limpia
// segmentos antes de buscar rutas, pero no emite esa redirección por sí solo.
func canonicalPath(c fiber.Ctx) error {
	raw := string(c.Request().URI().PathOriginal())
	clean := path.Clean("/" + strings.TrimPrefix(raw, "/"))
	if strings.HasSuffix(raw, "/") && clean != "/" {
		clean += "/"
	}
	if raw != clean {
		if c.Method() == fiber.MethodConnect {
			return notFound(c)
		}
		location := clean
		if query := c.Request().URI().QueryString(); len(query) > 0 {
			location += "?" + string(query)
		}
		c.Set(fiber.HeaderLocation, location)
		c.Status(fiber.StatusTemporaryRedirect)
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead {
			c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		}
		if c.Method() == fiber.MethodGet {
			return c.SendString("<a href=\"" + html.EscapeString(location) + "\">Temporary Redirect</a>.\n\n")
		}
		return c.Send(nil)
	}
	// ServeMux decodifica cada segmento después de limpiar la ruta escapada.
	// Un punto o separador escapado no debe convertir otra ruta en un endpoint.
	for _, segment := range strings.Split(raw, "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "." || decoded == ".." || strings.Contains(decoded, "/") {
			return notFound(c)
		}
	}
	return c.Next()
}
