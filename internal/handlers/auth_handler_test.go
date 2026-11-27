package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubUoW struct{ r repository.Repositories }

func (s stubUoW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, s.r)
}
func (s stubUoW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, s.r)
}
func (s stubUoW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, s.r)
}
func (s stubUoW) Repositories() repository.Repositories { return s.r }

type stubRepos struct {
	repository.Repositories
	personal  *stubPersonalRepo
	mfa       *stubMFARepo
	sesiones  *stubSesionRepo
	permisos  *stubPermisoRepo
	roles     *stubRolRepo
	intentos  *stubIntentosRepo
	auditoria *stubAuditRepo
}

func (r *stubRepos) Personal() repository.PersonalRepository                { return r.personal }
func (r *stubRepos) MFA() repository.MFAPersonalRepository                  { return r.mfa }
func (r *stubRepos) Sesiones() repository.SesionPersonalRepository          { return r.sesiones }
func (r *stubRepos) Permisos() repository.PermisoRepository                 { return r.permisos }
func (r *stubRepos) Roles() repository.RolRepository                        { return r.roles }
func (r *stubRepos) ControlIntentos() repository.ControlIntentosRepository  { return r.intentos }
func (r *stubRepos) EventosAuditoria() repository.EventoAuditoriaRepository { return r.auditoria }

type stubPersonalRepo struct {
	repository.PersonalRepository
	p *models.Personal
}

func (r *stubPersonalRepo) GetByUsuario(_ context.Context, u string) (*models.Personal, error) {
	if r.p != nil && r.p.Usuario == u {
		cp := *r.p
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (r *stubPersonalRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	if r.p != nil && r.p.ID == id {
		cp := *r.p
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (r *stubPersonalRepo) ReiniciarFallos(context.Context, uuid.UUID) error { return nil }
func (r *stubPersonalRepo) RegistrarFallo(context.Context, uuid.UUID, *time.Time) (int, *time.Time, error) {
	return 0, nil, nil
}

type stubMFARepo struct {
	repository.MFAPersonalRepository
}

func (r *stubMFARepo) GetByPersonalID(context.Context, uuid.UUID) (*models.MFAPersonal, error) {
	return nil, repository.ErrNotFound
}

type stubSesionRepo struct {
	repository.SesionPersonalRepository
	byID map[uuid.UUID]*models.SesionPersonal
}

func (r *stubSesionRepo) Create(_ context.Context, s *models.SesionPersonal) error {
	if r.byID == nil {
		r.byID = map[uuid.UUID]*models.SesionPersonal{}
	}
	cp := *s
	r.byID[s.ID] = &cp
	return nil
}
func (r *stubSesionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.SesionPersonal, error) {
	if s, ok := r.byID[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (r *stubSesionRepo) Revocar(_ context.Context, id uuid.UUID, at time.Time) error {
	if s, ok := r.byID[id]; ok {
		s.RevocadoEn = &at
		return nil
	}
	return repository.ErrNotFound
}
func (r *stubSesionRepo) ActualizarCSRF(_ context.Context, id uuid.UUID, hash []byte) error {
	if s, ok := r.byID[id]; ok {
		s.CSRFHash = hash
		return nil
	}
	return repository.ErrNotFound
}
func (r *stubSesionRepo) Touch(_ context.Context, id uuid.UUID, at time.Time) error {
	if s, ok := r.byID[id]; ok {
		s.UltimaActividad = at
	}
	return nil
}

type stubPermisoRepo struct{ repository.PermisoRepository }

func (r *stubPermisoRepo) CodigosEfectivos(context.Context, uuid.UUID, time.Time) ([]string, error) {
	return []string{"casefile.read"}, nil
}

type stubRolRepo struct{ repository.RolRepository }

func (r *stubRolRepo) GetByID(context.Context, uuid.UUID) (*models.Rol, error) {
	return &models.Rol{ID: uuid.New(), Codigo: "admin"}, nil
}

type stubIntentosRepo struct {
	repository.ControlIntentosRepository
}

func (r *stubIntentosRepo) EstaBloqueado(context.Context, string, []byte, time.Time) (bool, *time.Time, error) {
	return false, nil, nil
}
func (r *stubIntentosRepo) Incrementar(context.Context, string, []byte, time.Time) (*models.ControlIntentos, error) {
	return &models.ControlIntentos{Contador: 1}, nil
}
func (r *stubIntentosRepo) Bloquear(context.Context, string, []byte, time.Time, time.Time) error {
	return nil
}

type stubAuditRepo struct {
	repository.EventoAuditoriaRepository
}

func (r *stubAuditRepo) Append(context.Context, *models.EventoAuditoria) error { return nil }

func newAuthTestApp(t *testing.T) (*fiber.App, *stubSesionRepo) {
	t.Helper()
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	p := &models.Personal{ID: uuid.New(), Usuario: "admin", NombreLaboral: "Admin", RolID: uuid.New(), PasswordHash: hash, Estado: models.PersonalActivo}
	sesiones := &stubSesionRepo{byID: map[uuid.UUID]*models.SesionPersonal{}}
	repos := &stubRepos{
		personal:  &stubPersonalRepo{p: p},
		mfa:       &stubMFARepo{},
		sesiones:  sesiones,
		permisos:  &stubPermisoRepo{},
		roles:     &stubRolRepo{},
		intentos:  &stubIntentosRepo{},
		auditoria: &stubAuditRepo{},
	}
	svc, err := auth.NewService(stubUoW{r: repos}, "secreto-de-prueba-muy-largo-1234", time.Hour, bytes32())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h := NewAuthHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}
		return c.Status(code).JSON(fiber.Map{"error": err.Error()})
	}})
	authMw := middleware.AuthMiddleware("secreto-de-prueba-muy-largo-1234", stubUoW{r: repos})
	optionalAuth := func(c *fiber.Ctx) error {
		if c.Get(fiber.HeaderAuthorization) == "" && c.Cookies("sai_session") == "" {
			return c.Next()
		}
		return authMw(c)
	}
	app.Post("/api/auth/login", h.Login)
	app.Post("/api/auth/mfa/verify", optionalAuth, h.VerifyMFA)
	app.Get("/api/auth/csrf-token", optionalAuth, h.CSRFToken)
	protected := app.Group("", authMw, middleware.CSRF(stubUoW{r: repos}))
	protected.Post("/api/auth/logout", h.Logout)
	protected.Get("/api/auth/me", h.Me)
	return app, sesiones
}

func bytes32() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i + 1)
	}
	return b
}

func TestAuthLoginAcceptsUsernameAndSetsCookies(t *testing.T) {
	app, _ := newAuthTestApp(t)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["token"] == "" {
		t.Fatal("missing token")
	}
	cookies := resp.Header.Values("Set-Cookie")
	joined := strings.Join(cookies, ";")
	if !strings.Contains(joined, "sai_session=") || !strings.Contains(joined, "csrf_token=") {
		t.Fatalf("missing cookies: %v", cookies)
	}
}

func TestAuthCSRFTokentAnonymous(t *testing.T) {
	app, _ := newAuthTestApp(t)
	req := httptest.NewRequest("GET", "/api/auth/csrf-token", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["csrf_token"] == "" {
		t.Fatal("missing csrf_token")
	}
}

func TestAuthLogoutWithCookieAndCSRF(t *testing.T) {
	app, sesiones := newAuthTestApp(t)
	// login
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, _ := app.Test(loginReq)
	var loginBody map[string]any
	_ = json.NewDecoder(loginResp.Body).Decode(&loginBody)
	token, _ := loginBody["token"].(string)
	csrf, _ := loginBody["csrf_token"].(string)
	if token == "" || csrf == "" {
		t.Fatalf("missing login data: %+v", loginBody)
	}
	// find session cookie
	var sessionCookie string
	for _, c := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(c, "sai_session=") {
			sessionCookie = strings.SplitN(strings.SplitN(c, ";", 2)[0], "=", 2)[1]
		}
	}
	if sessionCookie == "" {
		t.Fatal("missing sai_session cookie")
	}

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Cookie", "sai_session="+sessionCookie)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 204 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	joined := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	if !strings.Contains(joined, "sai_session=;") {
		t.Fatalf("session cookie not cleared: %v", resp.Header.Values("Set-Cookie"))
	}
	for _, s := range sesiones.byID {
		if s.RevocadoEn == nil {
			t.Fatal("expected session to be revoked")
		}
	}
	_ = token
}
