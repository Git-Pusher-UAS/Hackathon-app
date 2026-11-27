package employer

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

type stubUOW struct {
	repository.UnitOfWork
	repos *stubRepos
}

func (u *stubUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, u.repos)
}
func (u *stubUOW) Repositories() repository.Repositories { return u.repos }

type stubRepos struct {
	repository.Repositories
	empleadores *stubEmpleadorRepo
	desafios    *stubDesafioRepo
	sesiones    *stubSesionRepo
}

func (r *stubRepos) Empleadores() repository.EmpleadorRepository              { return r.empleadores }
func (r *stubRepos) EmpleadorDesafios() repository.EmpleadorDesafioRepository { return r.desafios }
func (r *stubRepos) EmpleadorSesiones() repository.EmpleadorSesionRepository  { return r.sesiones }

type stubEmpleadorRepo struct {
	repository.EmpleadorRepository
	byCorreo map[string]*models.Empleador
}

func (r *stubEmpleadorRepo) GetByCorreo(_ context.Context, correo string) (*models.Empleador, error) {
	e, ok := r.byCorreo[correo]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

type stubDesafioRepo struct {
	repository.EmpleadorDesafioRepository
	items map[uuid.UUID]*models.EmpleadorDesafioOTP
}

func (r *stubDesafioRepo) Create(_ context.Context, d *models.EmpleadorDesafioOTP) error {
	cp := *d
	r.items[d.ID] = &cp
	return nil
}
func (r *stubDesafioRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmpleadorDesafioOTP, error) {
	d, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *d
	return &cp, nil
}
func (r *stubDesafioRepo) Save(_ context.Context, d *models.EmpleadorDesafioOTP) error {
	cp := *d
	r.items[d.ID] = &cp
	return nil
}

type stubSesionRepo struct {
	repository.EmpleadorSesionRepository
}

func (r *stubSesionRepo) Create(_ context.Context, _ *models.EmpleadorSesion) error { return nil }

func newEmployerTestFixture(devMode bool) (*service, string, *[]string) {
	emp := &models.Empleador{
		ID:       uuid.New(),
		Nombre:   "Empresa Demo",
		Correo:   "rrhh@empresa.mx",
		Telefono: "5511223344",
		Estado:   models.EmpleadorAprobado,
	}
	repos := &stubRepos{
		empleadores: &stubEmpleadorRepo{byCorreo: map[string]*models.Empleador{emp.Correo: emp}},
		desafios:    &stubDesafioRepo{items: map[uuid.UUID]*models.EmpleadorDesafioOTP{}},
		sesiones:    &stubSesionRepo{},
	}
	logs := []string{}
	svc := New(&stubUOW{repos: repos}, devMode, "test-otp-secret").(*service)
	svc.now = func() time.Time { return time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC) }
	svc.logf = func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	return svc, emp.Correo, &logs
}

func TestEmployerOTPLoggingRespectsDevMode(t *testing.T) {
	codeRe := regexp.MustCompile(`code=(\d{6})\b`)

	t.Run("logs working 6-digit OTP when devMode=true", func(t *testing.T) {
		svc, correo, logs := newEmployerTestFixture(true)

		ch, err := svc.Login(context.Background(), correo)
		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if len(*logs) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(*logs))
		}
		m := codeRe.FindStringSubmatch((*logs)[0])
		if len(m) != 2 {
			t.Fatalf("expected 6-digit code in log %q", (*logs)[0])
		}
		if _, err := svc.Verify(context.Background(), ch.ChallengeID, m[1]); err != nil {
			t.Fatalf("Verify with logged OTP failed: %v", err)
		}
	})

	t.Run("never logs OTP when devMode=false", func(t *testing.T) {
		svc, correo, logs := newEmployerTestFixture(false)

		if _, err := svc.Login(context.Background(), correo); err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if len(*logs) != 0 {
			t.Fatalf("expected 0 logs when devMode=false, got %v", *logs)
		}
	})
}

func TestEmployerLoginUnknownDoesNotEnumerate(t *testing.T) {
	svc, _, _ := newEmployerTestFixture(false)
	ch, err := svc.Login(context.Background(), "noexiste@empresa.mx")
	if err != nil {
		t.Fatalf("expected generic challenge, got err: %v", err)
	}
	if ch == nil || ch.ChallengeID == uuid.Nil {
		t.Fatalf("expected challenge response, got %+v", ch)
	}
}

func TestEmployerVerifyNonexistentChallenge(t *testing.T) {
	svc, _, _ := newEmployerTestFixture(false)
	_, err := svc.Verify(context.Background(), uuid.New(), "123456")
	if err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
