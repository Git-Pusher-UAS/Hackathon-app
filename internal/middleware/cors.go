package middleware

import (
	"net/http"
	"strings"
)

// CORS aplica los headers de CORS para el origen configurado.
//
// Si allowedOrigin es "*" se permite cualquier origen. Si es una lista separada
// por comas, solo se permite el origen exacto de la lista.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	allowed := strings.TrimSpace(allowedOrigin)
	allowAll := allowed == "*" || allowed == ""

	origins := make(map[string]bool)
	for _, o := range strings.Split(allowed, ",") {
		if o = strings.TrimSpace(o); o != "" && o != "*" {
			origins[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && origins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", HeaderRequestID)
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
