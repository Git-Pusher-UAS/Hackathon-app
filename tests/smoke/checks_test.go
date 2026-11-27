// Pruebas unitarias de las verificaciones de humo.
//
// Se ejecutan con `go test ./...` sin servidor real: levantan servidores falsos
// con httptest y comprueban que cada verificación aprueba un backend correcto
// y REPRUEBA uno defectuoso (el problema de los scripts de referencia eran los
// falsos positivos).
package smoke

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const validToken = "token-valido"

// goodServer se comporta como el backend esperado.
func goodServer() http.Handler {
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+validToken {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no autorizado"})
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /me", auth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"id": "1"})
	}))
	mux.HandleFunc("GET /items", auth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"data":       []int{1, 2},
			"pagination": map[string]int{"page": 1, "limit": 2, "total": 10},
		})
	}))
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["email"] == nil || body["password"] == nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "payload inválido"})
			return
		}
		writeJSON(w, http.StatusCreated, body)
	})
	return mux
}

func target(t *testing.T, h http.Handler) Target {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return Target{
		BaseURL:          srv.URL,
		HealthPath:       "/health",
		ProtectedRoutes:  []string{"/me"},
		ValidationRoutes: []string{"/register"},
		PaginatedRoutes:  []string{"/items"},
		Token:            validToken,
		MaxLatency:       time.Second,
		Client:           srv.Client(),
	}
}

func TestChecksPassOnGoodServer(t *testing.T) {
	tg := target(t, goodServer())
	for _, c := range All() {
		t.Run(c.Name, func(t *testing.T) {
			if err := c.Run(tg); err != nil {
				t.Fatalf("se esperaba aprobar: %v", err)
			}
		})
	}
}

// override reemplaza una ruta del servidor bueno por un comportamiento defectuoso.
func override(pattern string, h http.HandlerFunc) http.Handler {
	good := goodServer()
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, h)
	mux.Handle("/", good)
	return mux
}

func TestChecksFailOnBrokenServer(t *testing.T) {
	cases := []struct {
		name    string
		check   func(Target) error
		handler http.Handler
		modify  func(*Target)
	}{
		{
			name:    "health sin JSON",
			check:   CheckHealth,
			handler: override("GET /health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }),
		},
		{
			name:  "health con status incorrecto",
			check: CheckHealth,
			handler: override("GET /health", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"status":"degradado"}`))
			}),
		},
		{
			name:  "health lento",
			check: CheckHealthLatency,
			handler: override("GET /health", func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
				w.Write([]byte(`{"status":"ok"}`))
			}),
			modify: func(t *Target) { t.MaxLatency = 10 * time.Millisecond },
		},
		{
			name:  "ruta inexistente responde 200 (catch-all)",
			check: CheckUnknownRoute404,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"status":"ok"}`))
			}),
		},
		{
			name:  "ruta protegida abierta sin token",
			check: CheckProtectedWithoutToken,
			handler: override("GET /me", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"id":"1"}`))
			}),
		},
		{
			name:   "ruta protegida responde 404 en vez de 401",
			check:  CheckProtectedWithoutToken,
			modify: func(t *Target) { t.ProtectedRoutes = []string{"/no-existe"} },
		},
		{
			name:  "una de varias rutas protegidas está abierta",
			check: CheckProtectedWithoutToken,
			handler: override("GET /abierta", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{}`))
			}),
			modify: func(t *Target) { t.ProtectedRoutes = []string{"/me", "/abierta"} },
		},
		{
			name:  "acepta cualquier Bearer",
			check: CheckProtectedWithForgedToken,
			handler: override("GET /me", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					w.Write([]byte(`{"id":"1"}`))
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
			}),
		},
		{
			name:  "payload inválido provoca 500",
			check: CheckInvalidPayload,
			handler: override("POST /register", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
		},
		{
			name:  "payload inválido aceptado con 201",
			check: CheckInvalidPayload,
			handler: override("POST /register", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
		},
		{
			name:  "error 422 sin cuerpo estructurado",
			check: CheckInvalidPayload,
			handler: override("POST /register", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte("mal"))
			}),
		},
		{
			name:  "JSON malformado provoca 500",
			check: CheckMalformedJSON,
			handler: override("POST /register", func(w http.ResponseWriter, r *http.Request) {
				var v any
				if json.NewDecoder(r.Body).Decode(&v) != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}),
		},
		{
			name:  "500 bajo concurrencia",
			check: CheckConcurrency,
			handler: override("POST /register", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
		},
		{
			name:   "token válido rechazado",
			check:  CheckProtectedWithToken,
			modify: func(t *Target) { t.Token = "otro" },
		},
		{
			name:  "paginación sin metadatos",
			check: CheckPagination,
			handler: override("GET /items", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`[1,2]`))
			}),
		},
		{
			name:  "paginación ignora limit",
			check: CheckPagination,
			handler: override("GET /items", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"data":[1,2,3,4],"pagination":{"page":1,"limit":2,"total":4}}`))
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.handler
			if h == nil {
				h = goodServer()
			}
			tg := target(t, h)
			if tc.modify != nil {
				tc.modify(&tg)
			}
			err := tc.check(tg)
			var skip ErrSkip
			if err == nil || errors.As(err, &skip) {
				t.Fatalf("se esperaba que la verificación fallara, resultado: %v", err)
			}
		})
	}
}

func TestChecksSkipWithoutConfig(t *testing.T) {
	tg := target(t, goodServer())
	tg.Token = ""
	tg.ValidationRoutes = nil

	for _, check := range []func(Target) error{CheckProtectedWithToken, CheckPagination, CheckInvalidPayload, CheckMalformedJSON} {
		var skip ErrSkip
		if err := check(tg); !errors.As(err, &skip) {
			t.Errorf("se esperaba omitir, resultado: %v", err)
		}
	}
}
