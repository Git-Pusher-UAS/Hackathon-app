package user

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	usermiddleware "backend-hackathon/internal/middleware/user"
	usermodels "backend-hackathon/internal/models/user"
	repository "backend-hackathon/internal/repository/repositories"
	userservice "backend-hackathon/internal/services/user"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubUserService struct {
	userservice.Service
	meFn                func(ctx context.Context, actorID, sessionID uuid.UUID) (*usermodels.MeResponse, error)
	listMySessionsFn    func(ctx context.Context, actorID, currentSessionID uuid.UUID, page repository.PageRequest) (*usermodels.Page[usermodels.SessionSummary], error)
	revokeMySessionFn   func(ctx context.Context, actor repository.ActorContext, actorID, sessionID uuid.UUID) error
	revokeOthersFn      func(ctx context.Context, actor repository.ActorContext, actorID, currentSessionID uuid.UUID) error
	listFn              func(ctx context.Context, actorID uuid.UUID, q string, page repository.PageRequest) (*usermodels.Page[usermodels.StaffSummary], error)
	getFn               func(ctx context.Context, actorID, targetID uuid.UUID) (*usermodels.StaffSummary, error)
	createFn            func(ctx context.Context, actor repository.ActorContext, in usermodels.CreateRequest) (*usermodels.StaffSummary, error)
	updateFn            func(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, in usermodels.UpdateRequest, ifMatch string) (*usermodels.StaffSummary, error)
	revokeAllSessionsFn func(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string) error
	resetMFAFn          func(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string, verificacionPresencial bool) (uuid.UUID, error)
}

func (s *stubUserService) Me(ctx context.Context, actorID, sessionID uuid.UUID) (*usermodels.MeResponse, error) {
	return s.meFn(ctx, actorID, sessionID)
}

func (s *stubUserService) ListMySessions(ctx context.Context, actorID, currentSessionID uuid.UUID, page repository.PageRequest) (*usermodels.Page[usermodels.SessionSummary], error) {
	return s.listMySessionsFn(ctx, actorID, currentSessionID, page)
}

func (s *stubUserService) RevokeMySession(ctx context.Context, actor repository.ActorContext, actorID, sessionID uuid.UUID) error {
	return s.revokeMySessionFn(ctx, actor, actorID, sessionID)
}

func (s *stubUserService) RevokeOthers(ctx context.Context, actor repository.ActorContext, actorID, currentSessionID uuid.UUID) error {
	return s.revokeOthersFn(ctx, actor, actorID, currentSessionID)
}

func (s *stubUserService) List(ctx context.Context, actorID uuid.UUID, q string, page repository.PageRequest) (*usermodels.Page[usermodels.StaffSummary], error) {
	return s.listFn(ctx, actorID, q, page)
}

func (s *stubUserService) Get(ctx context.Context, actorID, targetID uuid.UUID) (*usermodels.StaffSummary, error) {
	return s.getFn(ctx, actorID, targetID)
}

func (s *stubUserService) Create(ctx context.Context, actor repository.ActorContext, in usermodels.CreateRequest) (*usermodels.StaffSummary, error) {
	return s.createFn(ctx, actor, in)
}

func (s *stubUserService) Update(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, in usermodels.UpdateRequest, ifMatch string) (*usermodels.StaffSummary, error) {
	return s.updateFn(ctx, actor, targetID, in, ifMatch)
}

func (s *stubUserService) RevokeAllSessions(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string) error {
	return s.revokeAllSessionsFn(ctx, actor, targetID, motivo)
}

func (s *stubUserService) ResetMFA(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string, verificacionPresencial bool) (uuid.UUID, error) {
	return s.resetMFAFn(ctx, actor, targetID, motivo, verificacionPresencial)
}

func newTestApp(svc userservice.Service, withAuth bool, actorID, sessionID uuid.UUID, csrfToken string) *fiber.App {
	return newTestAppWithPerms(svc, withAuth, actorID, sessionID, csrfToken, []string{"admin"})
}

func newTestAppWithPerms(svc userservice.Service, withAuth bool, actorID, sessionID uuid.UUID, csrfToken string, perms []string) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return httpx.Write(c, err)
		},
	})
	h := New(svc)

	apiGroup := app.Group("/api", func(c *fiber.Ctx) error {
		if withAuth {
			csrfHash := sha256.Sum256([]byte(csrfToken))
			c.Locals("personalID", actorID)
			c.Locals("sessionID", sessionID)
			c.Locals("csrfHash", csrfHash[:])
			c.Locals("permisos", perms)
			c.Locals("actor", repository.ActorContext{
				ActorID:   actorID,
				RequestID: middleware.RequestID(c),
			})
		}
		return c.Next()
	}, usermiddleware.NoStore)

	apiGroup.Get("/users/me", h.Me)
	apiGroup.Get("/users/me/sessions", h.ListMySessions)
	apiGroup.Delete("/users/me/sessions/:sessionId", usermiddleware.RequireCSRF, h.RevokeMySession)

	staff := apiGroup.Group("/users/staff", middleware.RequirePermiso("admin"))
	staff.Get("", h.List)
	staff.Post("", usermiddleware.RequireCSRF, h.Create)
	staff.Get("/:userId", h.Get)
	staff.Patch("/:userId", usermiddleware.RequireCSRF, h.Update)
	staff.Post("/:userId/force-logout", usermiddleware.RequireCSRF, h.ForceLogout)
	staff.Post("/:userId/session-revocations", usermiddleware.RequireCSRF, h.RevokeAllSessions)
	staff.Post("/:userId/mfa-resets", usermiddleware.RequireCSRF, h.ResetMFA)

	return app
}

func TestHandlerMe(t *testing.T) {
	actorID := uuid.New()
	sessionID := uuid.New()
	deptID := uuid.New()
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	t.Run("200 con headers no-store", func(t *testing.T) {
		svc := &stubUserService{
			meFn: func(_ context.Context, gotActor, gotSession uuid.UUID) (*usermodels.MeResponse, error) {
				if gotActor != actorID || gotSession != sessionID {
					t.Fatalf("ids = (%v,%v), esperado (%v,%v)", gotActor, gotSession, actorID, sessionID)
				}
				return &usermodels.MeResponse{
					Actor: usermodels.Actor{
						ID:           actorID,
						Rol:          "TrabajoSocial",
						DepartmentID: deptID,
						DisplayName:  "Ana López",
						Permissions:  []string{"case_files.read"},
					},
					ExpiresAt:     now.Add(10 * time.Hour),
					IdleExpiresAt: now.Add(15 * time.Minute),
				}, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("GET", "/api/users/me", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, esperado 200", resp.StatusCode)
		}
		if resp.Header.Get(fiber.HeaderCacheControl) != "no-store" || resp.Header.Get(fiber.HeaderPragma) != "no-cache" {
			t.Fatalf("headers caché inesperados: cache-control=%q pragma=%q", resp.Header.Get(fiber.HeaderCacheControl), resp.Header.Get(fiber.HeaderPragma))
		}

		var body usermodels.MeResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Actor.ID != actorID || body.Actor.Rol != "TrabajoSocial" {
			t.Fatalf("body inesperado: %+v", body)
		}
	})

	t.Run("401 sin autenticación", func(t *testing.T) {
		app := newTestApp(&stubUserService{}, false, actorID, sessionID, "")
		req := httptest.NewRequest("GET", "/api/users/me", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status = %d, esperado 401", resp.StatusCode)
		}
	})
}

func TestHandlerListMySessions(t *testing.T) {
	actorID := uuid.New()
	sessionID := uuid.New()

	t.Run("200 con paginación", func(t *testing.T) {
		next := "cursor-next"
		svc := &stubUserService{
			listMySessionsFn: func(_ context.Context, gotActor, gotSession uuid.UUID, page repository.PageRequest) (*usermodels.Page[usermodels.SessionSummary], error) {
				if gotActor != actorID || gotSession != sessionID {
					t.Fatalf("ids = (%v,%v)", gotActor, gotSession)
				}
				if page.Limit != 5 || page.Cursor != "c1" {
					t.Fatalf("page = %+v, esperado limit=5 cursor=c1", page)
				}
				return &usermodels.Page[usermodels.SessionSummary]{
					Items: []usermodels.SessionSummary{
						{ID: sessionID, DeviceLabel: "Navegador web", Current: true},
					},
					NextCursor: &next,
				}, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("GET", "/api/users/me/sessions?limit=5&cursor=c1", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, esperado 200", resp.StatusCode)
		}
	})

	t.Run("422 si limit está fuera de rango", func(t *testing.T) {
		app := newTestApp(&stubUserService{}, true, actorID, sessionID, "csrf-123")
		for _, target := range []string{
			"/api/users/me/sessions?limit=0",
			"/api/users/me/sessions?limit=101",
			"/api/users/me/sessions?limit=abc",
		} {
			resp, err := app.Test(httptest.NewRequest("GET", target, nil))
			if err != nil {
				t.Fatalf("app.Test(%s): %v", target, err)
			}
			if resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("%s status = %d, esperado 422", target, resp.StatusCode)
			}
		}
	})
}

func TestHandlerRevokeMySession(t *testing.T) {
	actorID := uuid.New()
	currentSessionID := uuid.New()
	targetSessionID := uuid.New()

	t.Run("403 sin X-CSRF-Token", func(t *testing.T) {
		app := newTestApp(&stubUserService{}, true, actorID, currentSessionID, "csrf-123")
		req := httptest.NewRequest("DELETE", "/api/users/me/sessions/"+targetSessionID.String(), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, esperado 403", resp.StatusCode)
		}
	})

	t.Run("422 con sessionId inválido", func(t *testing.T) {
		app := newTestApp(&stubUserService{}, true, actorID, currentSessionID, "csrf-123")
		req := httptest.NewRequest("DELETE", "/api/users/me/sessions/no-es-uuid", nil)
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusUnprocessableEntity {
			t.Fatalf("status = %d, esperado 422", resp.StatusCode)
		}
	})

	t.Run("404 si el servicio devuelve ErrNotFound", func(t *testing.T) {
		svc := &stubUserService{
			revokeMySessionFn: func(_ context.Context, _ repository.ActorContext, _, _ uuid.UUID) error {
				return userservice.ErrNotFound
			},
		}
		app := newTestApp(svc, true, actorID, currentSessionID, "csrf-123")
		req := httptest.NewRequest("DELETE", "/api/users/me/sessions/"+targetSessionID.String(), nil)
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("status = %d, esperado 404", resp.StatusCode)
		}
	})

	t.Run("500 si el servicio devuelve error inesperado", func(t *testing.T) {
		svc := &stubUserService{
			revokeMySessionFn: func(_ context.Context, _ repository.ActorContext, _, _ uuid.UUID) error {
				return errors.New("db down")
			},
		}
		app := newTestApp(svc, true, actorID, currentSessionID, "csrf-123")
		req := httptest.NewRequest("DELETE", "/api/users/me/sessions/"+targetSessionID.String(), nil)
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("status = %d, esperado 500", resp.StatusCode)
		}
	})

	t.Run("204 en revocación exitosa", func(t *testing.T) {
		svc := &stubUserService{
			revokeMySessionFn: func(_ context.Context, _ repository.ActorContext, gotActor, gotSession uuid.UUID) error {
				if gotActor != actorID || gotSession != targetSessionID {
					t.Fatalf("ids = (%v,%v)", gotActor, gotSession)
				}
				return nil
			},
		}
		app := newTestApp(svc, true, actorID, currentSessionID, "csrf-123")
		req := httptest.NewRequest("DELETE", "/api/users/me/sessions/"+targetSessionID.String(), nil)
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("status = %d, esperado 204", resp.StatusCode)
		}
	})
}

func TestHandlerAdminPermissionRequired(t *testing.T) {
	actorID := uuid.New()
	sessionID := uuid.New()
	targetID := uuid.New().String()
	app := newTestAppWithPerms(&stubUserService{}, true, actorID, sessionID, "csrf-123", []string{"case_files.read"})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{"GET", "/api/users/staff"},
		{"POST", "/api/users/staff"},
		{"PATCH", "/api/users/staff/" + targetID},
		{"POST", "/api/users/staff/" + targetID + "/session-revocations"},
		{"POST", "/api/users/staff/" + targetID + "/mfa-resets"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test(%s %s): %v", tc.method, tc.path, err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("%s %s status = %d, esperado 403", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestHandlerAdminListAndCreate(t *testing.T) {
	actorID := uuid.New()
	sessionID := uuid.New()
	deptID := uuid.New()

	t.Run("GET /users forwards query and page", func(t *testing.T) {
		svc := &stubUserService{
			listFn: func(_ context.Context, gotActor uuid.UUID, q string, page repository.PageRequest) (*usermodels.Page[usermodels.StaffSummary], error) {
				if gotActor != actorID || q != "ana" || page.Limit != 10 {
					t.Fatalf("args = (%v, %q, %+v)", gotActor, q, page)
				}
				return &usermodels.Page[usermodels.StaffSummary]{Items: []usermodels.StaffSummary{}}, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		resp, err := app.Test(httptest.NewRequest("GET", "/api/users/staff?q=ana&limit=10", nil))
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, esperado 200", resp.StatusCode)
		}
	})

	t.Run("POST /users returns 201 and ETag", func(t *testing.T) {
		svc := &stubUserService{
			createFn: func(_ context.Context, _ repository.ActorContext, in usermodels.CreateRequest) (*usermodels.StaffSummary, error) {
				if in.Usuario != "ana" {
					t.Fatalf("usuario = %q", in.Usuario)
				}
				return &usermodels.StaffSummary{
					ID:             uuid.New(),
					Usuario:        "ana",
					DepartamentoID: deptID,
					VersionTag:     "tag-v1",
				}, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("POST", "/api/users/staff", bytes.NewBufferString(`{"usuario":"ana","nombreLaboral":"Ana","rol":"TrabajoSocial","departamentoId":"`+deptID.String()+`","password":"Password1234"}`))
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("status = %d, esperado 201", resp.StatusCode)
		}
		if resp.Header.Get(fiber.HeaderETag) != `"tag-v1"` {
			t.Fatalf("ETag = %q, esperado %q", resp.Header.Get(fiber.HeaderETag), `"tag-v1"`)
		}
	})

	t.Run("POST /users rejects unknown JSON fields with 400", func(t *testing.T) {
		app := newTestApp(&stubUserService{}, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("POST", "/api/users/staff", bytes.NewBufferString(`{"extra":true}`))
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("status = %d, esperado 400", resp.StatusCode)
		}
	})
}

func TestHandlerAdminUpdateAndActions(t *testing.T) {
	actorID := uuid.New()
	sessionID := uuid.New()
	targetID := uuid.New()

	t.Run("PATCH /users/:userId requires If-Match (428) and returns ETag on 200", func(t *testing.T) {
		svc := &stubUserService{
			updateFn: func(_ context.Context, _ repository.ActorContext, gotTarget uuid.UUID, _ usermodels.UpdateRequest, ifMatch string) (*usermodels.StaffSummary, error) {
				if gotTarget != targetID || ifMatch != "tag-v1" {
					t.Fatalf("args = (%v, %q)", gotTarget, ifMatch)
				}
				return &usermodels.StaffSummary{ID: targetID, VersionTag: "tag-v2"}, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")

		reqNoMatch := httptest.NewRequest("PATCH", "/api/users/staff/"+targetID.String(), bytes.NewBufferString(`{"estado":"suspendido"}`))
		reqNoMatch.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(reqNoMatch)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusPreconditionRequired {
			t.Fatalf("status = %d, esperado 428", resp.StatusCode)
		}

		req := httptest.NewRequest("PATCH", "/api/users/staff/"+targetID.String(), bytes.NewBufferString(`{"estado":"suspendido"}`))
		req.Header.Set("X-CSRF-Token", "csrf-123")
		req.Header.Set(fiber.HeaderIfMatch, `"tag-v1"`)
		resp, err = app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK || resp.Header.Get(fiber.HeaderETag) != `"tag-v2"` {
			t.Fatalf("status=%d etag=%q", resp.StatusCode, resp.Header.Get(fiber.HeaderETag))
		}
	})

	t.Run("POST /users/:userId/session-revocations returns 204", func(t *testing.T) {
		svc := &stubUserService{
			revokeAllSessionsFn: func(_ context.Context, _ repository.ActorContext, gotTarget uuid.UUID, motivo string) error {
				if gotTarget != targetID || motivo != "motivo válido de prueba" {
					t.Fatalf("args = (%v, %q)", gotTarget, motivo)
				}
				return nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("POST", "/api/users/staff/"+targetID.String()+"/session-revocations", bytes.NewBufferString(`{"motivo":"motivo válido de prueba"}`))
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("status = %d, esperado 204", resp.StatusCode)
		}
	})

	t.Run("POST /users/:userId/mfa-resets returns 202 and resetId", func(t *testing.T) {
		resetID := uuid.New()
		svc := &stubUserService{
			resetMFAFn: func(_ context.Context, _ repository.ActorContext, gotTarget uuid.UUID, _ string, verified bool) (uuid.UUID, error) {
				if gotTarget != targetID || !verified {
					return uuid.Nil, userservice.ErrMFAVerificationRequired
				}
				return resetID, nil
			},
		}
		app := newTestApp(svc, true, actorID, sessionID, "csrf-123")
		req := httptest.NewRequest("POST", "/api/users/staff/"+targetID.String()+"/mfa-resets", bytes.NewBufferString(`{"motivo":"verificado en persona","verificacionPresencial":true}`))
		req.Header.Set("X-CSRF-Token", "csrf-123")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusAccepted {
			t.Fatalf("status = %d, esperado 202", resp.StatusCode)
		}
		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body["resetId"] != resetID.String() {
			t.Fatalf("body = %v (%v)", body, err)
		}
	})
}
