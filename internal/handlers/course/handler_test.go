package course_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/auth"
	coursehandler "backend-hackathon/internal/handlers/course"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	coursemw "backend-hackathon/internal/middleware/course"
	usermiddleware "backend-hackathon/internal/middleware/user"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	courseservice "backend-hackathon/internal/services/course"

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
	cursos         *stubCursoRepo
	inscripciones  *stubInscripcionRepo
	personal       *stubPersonalRepo
	sesiones       *stubSesionRepo
	portalSesiones *stubPortalSesionRepo
}

func (r *stubRepos) Cursos() repository.CursoRepository             { return r.cursos }
func (r *stubRepos) Inscripciones() repository.InscripcionRepository { return r.inscripciones }
func (r *stubRepos) Personal() repository.PersonalRepository        { return r.personal }
func (r *stubRepos) Sesiones() repository.SesionPersonalRepository  { return r.sesiones }
func (r *stubRepos) PortalSesiones() repository.PortalSesionRepository {
	return r.portalSesiones
}

type stubCursoRepo struct {
	repository.CursoRepository
	items  []models.Curso
	counts map[uuid.UUID]int
}

func (r *stubCursoRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Curso, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			cp := r.items[i]
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *stubCursoRepo) List(_ context.Context, _ repository.CursoFilter, _ repository.PageRequest) (repository.Page[models.Curso], error) {
	return repository.Page[models.Curso]{Items: r.items}, nil
}
func (r *stubCursoRepo) Inscritas(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return r.counts, nil
}

type stubInscripcionRepo struct {
	repository.InscripcionRepository
	enrolled map[uuid.UUID]map[uuid.UUID]bool
}

func (r *stubInscripcionRepo) GetByCursoAndUsuaria(_ context.Context, cursoID, usuariaID uuid.UUID) (*models.Inscripcion, error) {
	if r.enrolled[usuariaID][cursoID] {
		return &models.Inscripcion{ID: uuid.New(), CursoID: cursoID, UsuariaID: usuariaID}, nil
	}
	return nil, repository.ErrNotFound
}
func (r *stubInscripcionRepo) ExistsMany(_ context.Context, cursoIDs []uuid.UUID, usuariaID uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	for _, id := range cursoIDs {
		if r.enrolled[usuariaID][id] {
			out[id] = true
		}
	}
	return out, nil
}

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
	s *models.SesionPersonal
}

func (r *stubSesionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.SesionPersonal, error) {
	if r.s == nil || r.s.ID != id {
		return nil, repository.ErrNotFound
	}
	cp := *r.s
	return &cp, nil
}
func (r *stubSesionRepo) Touch(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }

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

func TestCoursesPortalAndStaffRoutes(t *testing.T) {
	const jwtSecret = "course-handler-test-secret"
	const portalSecret = "portal-tok-123"
	portalHash := sha256.Sum256([]byte(portalSecret))
	portalCuentaID := uuid.New()
	portalSessionID := uuid.New()

	cursoEnrolled := models.Curso{
		ID:          uuid.New(),
		Tipo:        models.CursoCurso,
		Titulo:      "Taller de finanzas",
		Descripcion: "Finanzas personales",
		Fecha:       time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		Hora:        "10:00",
		Lugar:       "Sala A",
		Cupo:        20,
		CreadoPor:   uuid.New(),
	}
	cursoOpen := models.Curso{
		ID:          uuid.New(),
		Tipo:        models.CursoCharla,
		Titulo:      "Derechos laborales",
		Descripcion: "Orientacion laboral",
		Fecha:       time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC),
		Hora:        "12:00",
		Lugar:       "Sala B",
		Cupo:        30,
		CreadoPor:   uuid.New(),
	}

	personalID := uuid.New()
	staffSessionID := uuid.New()

	makeStaff := func(perms []string) (string, *models.SesionPersonal) {
		tok, exp, err := auth.SignToken(jwtSecret, time.Hour, personalID, auth.Claims{
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

	setupApp := func(staffSes *models.SesionPersonal) *fiber.App {
		repos := &stubRepos{
			cursos: &stubCursoRepo{
				items:  []models.Curso{cursoEnrolled, cursoOpen},
				counts: map[uuid.UUID]int{cursoEnrolled.ID: 5, cursoOpen.ID: 2},
			},
			inscripciones: &stubInscripcionRepo{
				enrolled: map[uuid.UUID]map[uuid.UUID]bool{
					portalCuentaID: {cursoEnrolled.ID: true},
				},
			},
			personal: &stubPersonalRepo{
				p: &models.Personal{
					ID:              personalID,
					Estado:          models.PersonalActivo,
					VersionPermisos: 1,
				},
			},
			sesiones: &stubSesionRepo{s: staffSes},
			portalSesiones: &stubPortalSesionRepo{
				s: &models.PortalSesion{
					ID:              portalSessionID,
					CuentaID:        portalCuentaID,
					TokenHash:       portalHash[:],
					CSRFHash:        []byte{9, 9, 9},
					UltimaActividad: time.Now(),
					VenceAbsoluto:   time.Now().Add(time.Hour),
				},
			},
		}
		uow := &stubUOW{repos: repos}
		handler := coursehandler.New(courseservice.New(uow, jwtSecret))

		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				if strings.HasPrefix(c.Path(), "/api/") {
					return httpx.Write(c, err)
				}
				return c.SendStatus(fiber.StatusInternalServerError)
			},
		})
		api := app.Group("/api")
		courseReadAuth := middleware.PortalOrStaffAuthMiddleware(jwtSecret, uow, coursemw.PermManage)
		api.Get("/courses", courseReadAuth, usermiddleware.NoStore, handler.List)
		api.Get("/courses/:id", courseReadAuth, usermiddleware.NoStore, handler.Get)

		apiContract := app.Group("/api", middleware.AuthMiddleware(jwtSecret, uow), usermiddleware.NoStore)
		courses := apiContract.Group("/courses", coursemw.Require(coursemw.PermManage))
		courses.Post("", usermiddleware.RequireCSRF, handler.Create)
		courses.Get("/:id/attendance", handler.Attendance)
		courses.Put("/:id/attendance", usermiddleware.RequireCSRF, handler.SetAttendance)

		return app
	}

	portalCookie := middleware.PortalCookieName + "=" + portalSessionID.String() + "." + portalSecret

	t.Run("portal user gets 200 on GET /api/courses with inscrita populated and no-store", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/courses", nil)
		req.Header.Set("Cookie", portalCookie)
		resp, err := setupApp(nil).Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status=%d, want 200", resp.StatusCode)
		}
		if got := resp.Header.Get(fiber.HeaderCacheControl); got != "no-store" {
			t.Fatalf("Cache-Control=%q, want no-store", got)
		}
		body, _ := io.ReadAll(resp.Body)
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("len=%d, want 2", len(items))
		}
		byID := map[string]map[string]any{}
		for _, it := range items {
			byID[it["id"].(string)] = it
		}
		if got, ok := byID[cursoEnrolled.ID.String()]["inscrita"].(bool); !ok || !got {
			t.Fatalf("enrolled course inscrita=%v (ok=%v), want true", byID[cursoEnrolled.ID.String()]["inscrita"], ok)
		}
		if got, ok := byID[cursoOpen.ID.String()]["inscrita"].(bool); !ok || got {
			t.Fatalf("open course inscrita=%v (ok=%v), want false", byID[cursoOpen.ID.String()]["inscrita"], ok)
		}
	})

	t.Run("portal user gets 200 on GET /api/courses/:id with inscrita true/false", func(t *testing.T) {
		app := setupApp(nil)

		for _, tc := range []struct {
			id   uuid.UUID
			want bool
		}{
			{id: cursoEnrolled.ID, want: true},
			{id: cursoOpen.ID, want: false},
		} {
			req := httptest.NewRequest("GET", "/api/courses/"+tc.id.String(), nil)
			req.Header.Set("Cookie", portalCookie)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != fiber.StatusOK {
				t.Fatalf("id=%s status=%d, want 200", tc.id, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			var item map[string]any
			if err := json.Unmarshal(body, &item); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, ok := item["inscrita"].(bool)
			if !ok || got != tc.want {
				t.Fatalf("id=%s inscrita=%v (ok=%v), want %v", tc.id, item["inscrita"], ok, tc.want)
			}
		}
	})

	t.Run("staff with courses.manage gets 200 with inscrita omitted", func(t *testing.T) {
		tok, ses := makeStaff([]string{coursemw.PermManage})
		app := setupApp(ses)

		reqList := httptest.NewRequest("GET", "/api/courses", nil)
		reqList.Header.Set(fiber.HeaderAuthorization, "Bearer "+tok)
		respList, err := app.Test(reqList)
		if err != nil {
			t.Fatal(err)
		}
		if respList.StatusCode != fiber.StatusOK {
			t.Fatalf("list status=%d, want 200", respList.StatusCode)
		}
		listBody, _ := io.ReadAll(respList.Body)
		var items []map[string]any
		if err := json.Unmarshal(listBody, &items); err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			if _, exists := it["inscrita"]; exists {
				t.Fatalf("expected inscrita omitted for staff, got %v", it["inscrita"])
			}
		}

		reqGet := httptest.NewRequest("GET", "/api/courses/"+cursoOpen.ID.String(), nil)
		reqGet.Header.Set(fiber.HeaderAuthorization, "Bearer "+tok)
		respGet, err := app.Test(reqGet)
		if err != nil {
			t.Fatal(err)
		}
		if respGet.StatusCode != fiber.StatusOK {
			t.Fatalf("get status=%d, want 200", respGet.StatusCode)
		}
		getBody, _ := io.ReadAll(respGet.Body)
		var item map[string]any
		if err := json.Unmarshal(getBody, &item); err != nil {
			t.Fatal(err)
		}
		if _, exists := item["inscrita"]; exists {
			t.Fatalf("expected inscrita omitted for staff get, got %v", item["inscrita"])
		}
	})

	t.Run("staff without courses.manage gets 403 on course GET routes", func(t *testing.T) {
		tok, ses := makeStaff([]string{"case_files.read"})
		app := setupApp(ses)

		for _, path := range []string{"/api/courses", "/api/courses/" + cursoOpen.ID.String()} {
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set(fiber.HeaderAuthorization, "Bearer "+tok)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != fiber.StatusForbidden {
				t.Fatalf("path=%s status=%d, want 403", path, resp.StatusCode)
			}
		}
	})

	t.Run("unauthenticated gets 401 and portal gets 401 on staff-only course routes", func(t *testing.T) {
		app := setupApp(nil)

		for _, path := range []string{"/api/courses", "/api/courses/" + cursoOpen.ID.String()} {
			resp, err := app.Test(httptest.NewRequest("GET", path, nil))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("unauth path=%s status=%d, want 401", path, resp.StatusCode)
			}
		}

		for _, tc := range []struct {
			method string
			path   string
		}{
			{"POST", "/api/courses"},
			{"GET", "/api/courses/" + cursoOpen.ID.String() + "/attendance"},
			{"PUT", "/api/courses/" + cursoOpen.ID.String() + "/attendance"},
		} {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Cookie", portalCookie)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("portal %s %s status=%d, want 401", tc.method, tc.path, resp.StatusCode)
			}
		}
	})
}
