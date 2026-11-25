package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"backend-hackathon/internal/httpx"
)

// Recover convierte un panic en un 500 JSON en lugar de tumbar el proceso.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic en %s %s (request_id=%s): %v\n%s",
					r.Method, r.URL.Path, RequestIDFrom(r.Context()), rec, debug.Stack())
				httpx.Error(w, http.StatusInternalServerError, "internal_error", "error interno del servidor")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
