package server

import (
	"net/http"

	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
)

// router registra las rutas y encadena los middlewares.
//
// Orden (de fuera hacia dentro): Recover → RequestID → Logging → CORS → rutas.
func router(cfg *config.Config, db handlers.Pinger, version string) http.Handler {
	status := handlers.NewStatus(db, version)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", status.Health)
	mux.HandleFunc("GET /api/v1/status", status.Status)

	// Patrones sin método: reciben los métodos que no coinciden con "GET ...".
	// ServeMux prefiere el patrón con método para GET y cae aquí para el resto.
	notAllowed := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, OPTIONS")
		httpx.Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "método no permitido")
	}
	mux.HandleFunc("/health", notAllowed)
	mux.HandleFunc("/api/v1/status", notAllowed)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusNotFound, "not_found", "ruta no encontrada")
	})

	var h http.Handler = mux
	h = middleware.CORS(cfg.Server.CORSOrigin)(h)
	h = middleware.Logging(h)
	h = middleware.RequestID(h)
	h = middleware.Recover(h)

	return h
}
