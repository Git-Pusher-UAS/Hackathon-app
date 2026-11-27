// Package smoke contiene las verificaciones de humo (smoke tests) del backend.
//
// Cada verificación es una función que recibe un Target y devuelve nil si pasa
// o un error que explica el fallo. Así las mismas verificaciones se usan:
//   - contra un servidor real   (smoke_test.go, build tag "smoke")
//   - contra servidores falsos  (checks_test.go, pruebas unitarias normales)
//
// A diferencia de los scripts de referencia (C:\23124067\scripts), aquí:
//   - cada ruta debe cumplir por sí misma (no basta con que pase la primera),
//   - un 404 nunca cuenta como éxito en una ruta que debe existir,
//   - no hay umbral de puntos: un solo fallo hace fallar la suite.
package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Target describe el servidor a probar y qué rutas cubre cada verificación.
type Target struct {
	BaseURL          string
	HealthPath       string
	ProtectedRoutes  []string      // GET que exigen JWT
	ValidationRoutes []string      // POST que validan el cuerpo
	PaginatedRoutes  []string      // GET que devuelven listas paginadas
	Token            string        // JWT válido opcional, habilita las verificaciones autenticadas
	MaxLatency       time.Duration // tiempo máximo aceptable del healthcheck
	Client           *http.Client
}

// TargetFromEnv construye un Target a partir de variables de entorno.
// Los valores por defecto corresponden a las rutas de cmd/main.go.
func TargetFromEnv() Target {
	return Target{
		BaseURL:          env("SMOKE_BASE_URL", "http://localhost:"+env("PORT", "3000")),
		HealthPath:       env("SMOKE_HEALTH_PATH", "/health"),
		ProtectedRoutes:  list("SMOKE_PROTECTED_ROUTES", "/api/v1/auth/me,/api/v1/users/me"),
		ValidationRoutes: list("SMOKE_VALIDATION_ROUTES", "/api/v1/auth/login"),
		PaginatedRoutes:  list("SMOKE_PAGINATED_ROUTES", "/api/v1/case-files"),
		Token:            os.Getenv("SMOKE_TOKEN"),
		MaxLatency:       500 * time.Millisecond,
		Client:           &http.Client{Timeout: 5 * time.Second},
	}
}

// Check es una verificación con nombre.
type Check struct {
	Name string
	Run  func(Target) error
}

// All devuelve las verificaciones en el orden en que se ejecutan.
func All() []Check {
	return []Check{
		{"health_responde_200_con_status_ok", CheckHealth},
		{"health_bajo_latencia_maxima", CheckHealthLatency},
		{"ruta_inexistente_responde_404", CheckUnknownRoute404},
		{"rutas_protegidas_sin_token_responden_401", CheckProtectedWithoutToken},
		{"rutas_protegidas_con_jwt_falso_responden_401", CheckProtectedWithForgedToken},
		{"payload_invalido_rechazado_con_4xx_estructurado", CheckInvalidPayload},
		{"json_malformado_no_provoca_500", CheckMalformedJSON},
		{"peticiones_concurrentes_sin_5xx", CheckConcurrency},
		{"rutas_protegidas_con_token_valido_responden_200", CheckProtectedWithToken},
		{"listas_paginadas_cumplen_contrato", CheckPagination},
	}
}

// ErrSkip indica que la verificación no aplica con la configuración actual.
type ErrSkip struct{ Reason string }

func (e ErrSkip) Error() string { return "omitida: " + e.Reason }

type response struct {
	status  int
	body    []byte
	elapsed time.Duration
}

func (t Target) do(method, path, token string, body []byte) (response, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, strings.TrimRight(t.BaseURL, "/")+path, r)
	if err != nil {
		return response{}, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	start := time.Now()
	res, err := client.Do(req)
	if err != nil {
		return response{}, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return response{}, fmt.Errorf("%s %s: leyendo cuerpo: %w", method, path, err)
	}
	return response{status: res.StatusCode, body: data, elapsed: time.Since(start)}, nil
}

func CheckHealth(t Target) error {
	res, err := t.do(http.MethodGet, t.HealthPath, "", nil)
	if err != nil {
		return err
	}
	if res.status != http.StatusOK {
		return fmt.Errorf("GET %s: se esperaba 200, se recibió %d", t.HealthPath, res.status)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.body, &payload); err != nil {
		return fmt.Errorf("GET %s: la respuesta no es JSON: %q", t.HealthPath, truncate(res.body))
	}
	status, _ := payload["status"].(string)
	switch strings.ToLower(status) {
	case "ok", "up", "healthy":
		return nil
	}
	return fmt.Errorf("GET %s: campo status esperado ok/up/healthy, se recibió %q", t.HealthPath, status)
}

func CheckHealthLatency(t Target) error {
	res, err := t.do(http.MethodGet, t.HealthPath, "", nil)
	if err != nil {
		return err
	}
	if t.MaxLatency > 0 && res.elapsed > t.MaxLatency {
		return fmt.Errorf("GET %s tardó %s (máximo %s)", t.HealthPath, res.elapsed, t.MaxLatency)
	}
	return nil
}

func CheckUnknownRoute404(t Target) error {
	const path = "/__ruta_que_no_existe_smoke__"
	res, err := t.do(http.MethodGet, path, "", nil)
	if err != nil {
		return err
	}
	if res.status != http.StatusNotFound {
		return fmt.Errorf("GET %s: se esperaba 404, se recibió %d", path, res.status)
	}
	return nil
}

func CheckProtectedWithoutToken(t Target) error {
	return expectAll(t, t.ProtectedRoutes, "", http.StatusUnauthorized)
}

// JWT con estructura válida pero firma inventada.
const forgedJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIn0.firma_falsa"

func CheckProtectedWithForgedToken(t Target) error {
	if err := expectAll(t, t.ProtectedRoutes, forgedJWT, http.StatusUnauthorized); err != nil {
		return err
	}
	// alg "none": un clásico que algunas librerías aceptan si no se fija el algoritmo.
	const noneJWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIn0."
	return expectAll(t, t.ProtectedRoutes, noneJWT, http.StatusUnauthorized)
}

func CheckProtectedWithToken(t Target) error {
	if t.Token == "" {
		return ErrSkip{"define SMOKE_TOKEN para probar rutas autenticadas"}
	}
	return expectAll(t, t.ProtectedRoutes, t.Token, http.StatusOK)
}

// CheckInvalidPayload exige que un cuerpo sin los campos requeridos se rechace
// con 400/422, o 401 en endpoints de login (que no revelan si el usuario existe),
// y con un error en JSON. Nunca 2xx ni 5xx.
func CheckInvalidPayload(t Target) error {
	if len(t.ValidationRoutes) == 0 {
		return ErrSkip{"no hay rutas de validación configuradas"}
	}
	var errs []string
	for _, path := range t.ValidationRoutes {
		res, err := t.do(http.MethodPost, path, "", []byte(`{"campo_inesperado":"x"}`))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if res.status != http.StatusBadRequest && res.status != http.StatusUnauthorized && res.status != http.StatusUnprocessableEntity {
			errs = append(errs, fmt.Sprintf("POST %s: se esperaba 400/401/422, se recibió %d", path, res.status))
			continue
		}
		var payload map[string]any
		if json.Unmarshal(res.body, &payload) != nil || !hasAnyKey(payload, "error", "errors", "message", "detail", "title") {
			errs = append(errs, fmt.Sprintf("POST %s: el error no es JSON estructurado: %q", path, truncate(res.body)))
		}
	}
	return joinErrs(errs)
}

func CheckMalformedJSON(t Target) error {
	if len(t.ValidationRoutes) == 0 {
		return ErrSkip{"no hay rutas de validación configuradas"}
	}
	var errs []string
	for _, path := range t.ValidationRoutes {
		res, err := t.do(http.MethodPost, path, "", []byte(`{"email": `))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if res.status >= 500 {
			errs = append(errs, fmt.Sprintf("POST %s con JSON roto devolvió %d", path, res.status))
		}
	}
	return joinErrs(errs)
}

func CheckConcurrency(t Target) error {
	const n = 20
	paths := append([]string{t.HealthPath}, t.ValidationRoutes...)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []string
	)
	for i := 0; i < n; i++ {
		path := paths[i%len(paths)]
		method, body := http.MethodGet, []byte(nil)
		if path != t.HealthPath {
			method, body = http.MethodPost, []byte(fmt.Sprintf(`{"email":"smoke%d@test.local"}`, i))
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := t.do(method, path, "", body)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err.Error())
			} else if res.status >= 500 {
				errs = append(errs, fmt.Sprintf("%s %s devolvió %d bajo concurrencia", method, path, res.status))
			}
		}()
	}
	wg.Wait()
	return joinErrs(errs)
}

func CheckPagination(t Target) error {
	if len(t.PaginatedRoutes) == 0 {
		return ErrSkip{"define SMOKE_PAGINATED_ROUTES"}
	}
	if t.Token == "" {
		return ErrSkip{"define SMOKE_TOKEN; sin token solo se probaría el 401"}
	}
	var errs []string
	for _, path := range t.PaginatedRoutes {
		full := path + sep(path) + "limit=2"
		res, err := t.do(http.MethodGet, full, t.Token, nil)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if res.status != http.StatusOK {
			errs = append(errs, fmt.Sprintf("GET %s: se esperaba 200, se recibió %d", full, res.status))
			continue
		}
		// Contrato §1.6: {"items": [...], "nextCursor": "..." | null}.
		var payload struct {
			Items      []json.RawMessage `json:"items"`
			NextCursor *json.RawMessage  `json:"nextCursor"`
		}
		var keys map[string]json.RawMessage
		_ = json.Unmarshal(res.body, &keys)
		_, hasCursor := keys["nextCursor"]
		if err := json.Unmarshal(res.body, &payload); err != nil || payload.Items == nil || !hasCursor {
			errs = append(errs, fmt.Sprintf("GET %s: se esperaba {items: [], nextCursor}, se recibió %q", full, truncate(res.body)))
			continue
		}
		if len(payload.Items) > 2 {
			errs = append(errs, fmt.Sprintf("GET %s: limit=2 pero devolvió %d elementos", full, len(payload.Items)))
		}
	}
	return joinErrs(errs)
}

// expectAll exige que TODAS las rutas devuelvan exactamente el código esperado.
func expectAll(t Target, paths []string, token string, want int) error {
	if len(paths) == 0 {
		return ErrSkip{"no hay rutas configuradas"}
	}
	var errs []string
	for _, path := range paths {
		res, err := t.do(http.MethodGet, path, token, nil)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if res.status != want {
			errs = append(errs, fmt.Sprintf("GET %s: se esperaba %d, se recibió %d", path, want, res.status))
		}
	}
	return joinErrs(errs)
}

func joinErrs(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "\n"))
}

func hasAnyKey(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

func truncate(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "..."
	}
	return string(b)
}

func sep(path string) string {
	if strings.Contains(path, "?") {
		return "&"
	}
	return "?"
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func list(key, def string) []string {
	var out []string
	for _, p := range strings.Split(env(key, def), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
