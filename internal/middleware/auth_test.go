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

type stubUOW struct {
	repository.UnitOfWork
	repos *stubRepos
}

func (u *stubUOW) Repositories() repository.Repositories { return u.repos }

type stubRepos struct {
	repository.Repositories
	personal *stubPersonalRepo
	sesiones *stubSesionRepo
}

func (r *stubRepos) Personal() repository.PersonalRepository     { return r.personal }
func (r *stubRepos) Sesiones() repository.SesionPersonalRepository { return r.sesiones }

type stubPersonalRepo struct {
	repository.PersonalRepository
	p *models.Personal
}

func (r *stubPersonalRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	if r.p == nil || r.p.ID != id {
		return nil, repository.ErrNotFound
	}
	cp := *r.p
	return &cp, nil
}

type stubSesionRepo struct {
	repository.SesionPersonalRepository
	s       *models.SesionPersonal
	touched bool
}

func (r *stubSesionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.SesionPersonal, error) {
	if r.s == nil || r.s.ID != id {
		return nil, repository.ErrNotFound
	}
	cp := *r.s
	return &cp, nil
}

func (r *stubSesionRepo) Touch(_ context.Context, _ uuid.UUID, _ time.Time) error {
	r.touched = true
	return nil
}

func TestAuthMiddlewareIdleAndSessionChecks(t *testing.T) {
	secret := "secreto-middleware"
	personalID := uuid.New()
	sessionID := uuid.New()

	token, expires, err := auth.SignToken(secret, time.Hour, personalID, auth.Claims{
		ActorID:     personalID,
		SessionID:   sessionID,
		Rol:         "admin",
		Permisos:    []string{"admin"},
		PermVersion: 1,
	})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	tokenHash := sha256.Sum256([]byte(token))

	runRequest := func(sesion *models.SesionPersonal) (int, bool) {
		repos := &stubRepos{
			personal: &stubPersonalRepo{
				p: &models.Personal{
					ID:              personalID,
					Estado:          models.PersonalActivo,
					VersionPermisos: 1,
				},
			},
			sesiones: &stubSesionRepo{s: sesion},
		}
		app := fiber.New()
		app.Use(AuthMiddleware(secret, &stubUOW{repos: repos}))
		app.Get("/protected", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		return resp.StatusCode, repos.sesiones.touched
	}

	t.Run("acepta sesión activa y actualiza actividad si >1m", func(t *testing.T) {
		code, touched := runRequest(&models.SesionPersonal{
			ID:              sessionID,
			TitularID:       personalID,
			TokenHash:       tokenHash[:],
			CSRFHash:        []byte{1, 2, 3},
			UltimaActividad: time.Now().Add(-2 * time.Minute),
			VenceAbsoluto:   expires,
			VersionPermisos: 1,
		})
		if code != fiber.StatusOK || !touched {
			t.Fatalf("status=%d touched=%v, esperado 200 y touched=true", code, touched)
		}
	})

	t.Run("rechaza sesión vencida por inactividad", func(t *testing.T) {
		code, touched := runRequest(&models.SesionPersonal{
			ID:              sessionID,
			TitularID:       personalID,
			TokenHash:       tokenHash[:],
			CSRFHash:        []byte{1, 2, 3},
			UltimaActividad: time.Now().Add(-(auth.IdleTimeout + time.Second)),
			VenceAbsoluto:   expires,
			VersionPermisos: 1,
		})
		if code != fiber.StatusUnauthorized || touched {
			t.Fatalf("status=%d touched=%v, esperado 401 y touched=false", code, touched)
		}
	})
}
