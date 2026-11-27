package middleware

import (
	"context"
	"crypto/sha256"
	"net/http/httptest"
	"testing"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type combinedRepos struct {
	repository.Repositories
	personal       *stubPersonalRepo
	sesiones       *stubSesionRepo
	portalSesiones *stubPortalSesionRepo
}

func (r *combinedRepos) Personal() repository.PersonalRepository             { return r.personal }
func (r *combinedRepos) Sesiones() repository.SesionPersonalRepository       { return r.sesiones }
func (r *combinedRepos) PortalSesiones() repository.PortalSesionRepository   { return r.portalSesiones }

type combinedUOW struct {
	repository.UnitOfWork
	repos *combinedRepos
}

func (u *combinedUOW) Repositories() repository.Repositories { return u.repos }

type stubPortalSesionRepo struct {
	repository.PortalSesionRepository
	s *models.PortalSesion
}

func (r *stubPortalSesionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PortalSesion, error) {
	if r.s == nil || r.s.ID != id {
		return nil, repository.ErrNotFound
	}
	cp := *r.s
	return &cp, nil
}
func (r *stubPortalSesionRepo) Touch(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }

func TestPortalOrStaffAuthMiddleware(t *testing.T) {
	const secret = "test-secret-courses"
	const portalToken = "portal-secret-token"
	portalTokenHash := sha256.Sum256([]byte(portalToken))
	portalCuentaID := uuid.New()
	portalSessionID := uuid.New()

	personalID := uuid.New()
	staffSessionID := uuid.New()

	makeStaffToken := func(perms []string) (string, *models.SesionPersonal) {
		tok, exp, err := auth.SignToken(secret, time.Hour, personalID, auth.Claims{
			ActorID:     personalID,
			SessionID:   staffSessionID,
			Rol:         "staff",
			Permisos:    perms,
			PermVersion: 1,
		})
		if err != nil {
			t.Fatalf("SignToken: %v", err)
		}
		h := sha256.Sum256([]byte(tok))
		return tok, &models.SesionPersonal{
			ID:              staffSessionID,
			TitularID:       personalID,
			TokenHash:       h[:],
			CSRFHash:        []byte{1, 2, 3},
			UltimaActividad: time.Now(),
			VenceAbsoluto:   exp,
			VersionPermisos: 1,
		}
	}

	activePortalSession := &models.PortalSesion{
		ID:              portalSessionID,
		CuentaID:        portalCuentaID,
		TokenHash:       portalTokenHash[:],
		CSRFHash:        []byte{4, 5, 6},
		UltimaActividad: time.Now(),
		VenceAbsoluto:   time.Now().Add(time.Hour),
	}

	newApp := func(portalSes *models.PortalSesion, staffSes *models.SesionPersonal) *fiber.App {
		repos := &combinedRepos{
			personal: &stubPersonalRepo{
				p: &models.Personal{
					ID:              personalID,
					Estado:          models.PersonalActivo,
					VersionPermisos: 1,
				},
			},
			sesiones:       &stubSesionRepo{s: staffSes},
			portalSesiones: &stubPortalSesionRepo{s: portalSes},
		}
		app := fiber.New()
		app.Get("/courses", PortalOrStaffAuthMiddleware(secret, &combinedUOW{repos: repos}, "courses.manage"), func(c *fiber.Ctx) error {
			pid, isPortal := PortalCuentaID(c)
			actor, hasActor := ActorFrom(c)
			if !hasActor {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			if isPortal && pid != portalCuentaID {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			if isPortal && actor.ActorID != portalCuentaID {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			if isPortal {
				return c.SendString("portal")
			}
			return c.SendString("staff")
		})
		return app
	}

	t.Run("rejects unauthenticated request with 401", func(t *testing.T) {
		resp, err := newApp(activePortalSession, nil).Test(httptest.NewRequest("GET", "/courses", nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", resp.StatusCode)
		}
	})

	t.Run("allows valid portal cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/courses", nil)
		req.Header.Set("Cookie", PortalCookieName+"="+portalSessionID.String()+"."+portalToken)
		resp, err := newApp(activePortalSession, nil).Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status=%d, want 200", resp.StatusCode)
		}
	})

	t.Run("rejects expired or revoked portal cookie with 401", func(t *testing.T) {
		revokedAt := time.Now().Add(-time.Minute)
		revoked := *activePortalSession
		revoked.RevocadoEn = &revokedAt

		req := httptest.NewRequest("GET", "/courses", nil)
		req.Header.Set("Cookie", PortalCookieName+"="+portalSessionID.String()+"."+portalToken)
		resp, err := newApp(&revoked, nil).Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", resp.StatusCode)
		}
	})

	t.Run("allows staff Bearer with courses.manage and admin", func(t *testing.T) {
		for _, perm := range []string{"courses.manage", "admin"} {
			tok, ses := makeStaffToken([]string{perm})
			req := httptest.NewRequest("GET", "/courses", nil)
			req.Header.Set(fiber.HeaderAuthorization, "Bearer "+tok)
			resp, err := newApp(nil, ses).Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != fiber.StatusOK {
				t.Fatalf("perm=%s status=%d, want 200", perm, resp.StatusCode)
			}
		}
	})

	t.Run("rejects staff Bearer without courses.manage with 403", func(t *testing.T) {
		tok, ses := makeStaffToken([]string{"case_files.read"})
		req := httptest.NewRequest("GET", "/courses", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+tok)
		resp, err := newApp(nil, ses).Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status=%d, want 403", resp.StatusCode)
		}
	})

	t.Run("handles staff sai_session cookie with and without permission", func(t *testing.T) {
		tokOK, sesOK := makeStaffToken([]string{"courses.manage"})
		reqOK := httptest.NewRequest("GET", "/courses", nil)
		reqOK.Header.Set("Cookie", "sai_session="+tokOK)
		respOK, err := newApp(nil, sesOK).Test(reqOK)
		if err != nil {
			t.Fatal(err)
		}
		if respOK.StatusCode != fiber.StatusOK {
			t.Fatalf("status=%d, want 200", respOK.StatusCode)
		}

		tokNo, sesNo := makeStaffToken([]string{"case_files.read"})
		reqNo := httptest.NewRequest("GET", "/courses", nil)
		reqNo.Header.Set("Cookie", "sai_session="+tokNo)
		respNo, err := newApp(nil, sesNo).Test(reqNo)
		if err != nil {
			t.Fatal(err)
		}
		if respNo.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status=%d, want 403", respNo.StatusCode)
		}
	})

	t.Run("explicit Bearer takes precedence over portal cookie and falls through from expired portal cookie to staff cookie", func(t *testing.T) {
		tokNo, sesNo := makeStaffToken([]string{"case_files.read"})
		reqBearer := httptest.NewRequest("GET", "/courses", nil)
		reqBearer.Header.Set(fiber.HeaderAuthorization, "Bearer "+tokNo)
		reqBearer.Header.Set("Cookie", PortalCookieName+"="+portalSessionID.String()+"."+portalToken)
		respBearer, err := newApp(activePortalSession, sesNo).Test(reqBearer)
		if err != nil {
			t.Fatal(err)
		}
		if respBearer.StatusCode != fiber.StatusForbidden {
			t.Fatalf("explicit Bearer without perm: status=%d, want 403", respBearer.StatusCode)
		}

		expired := *activePortalSession
		expired.VenceAbsoluto = time.Now().Add(-time.Minute)
		tokOK, sesOK := makeStaffToken([]string{"courses.manage"})
		reqFallback := httptest.NewRequest("GET", "/courses", nil)
		reqFallback.Header.Set("Cookie", PortalCookieName+"="+portalSessionID.String()+"."+portalToken+"; sai_session="+tokOK)
		respFallback, err := newApp(&expired, sesOK).Test(reqFallback)
		if err != nil {
			t.Fatal(err)
		}
		if respFallback.StatusCode != fiber.StatusOK {
			t.Fatalf("expired portal + valid staff cookie: status=%d, want 200", respFallback.StatusCode)
		}
	})
}
