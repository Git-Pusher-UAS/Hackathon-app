package portal

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	pmodels "backend-hackathon/internal/models/portal"
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
	cuentas  *stubCuentaRepo
	desafios *stubDesafioRepo
	sesiones *stubSesionRepo
}

func (r *stubRepos) PortalCuentas() repository.PortalCuentaRepository   { return r.cuentas }
func (r *stubRepos) PortalDesafios() repository.PortalDesafioRepository { return r.desafios }
func (r *stubRepos) PortalSesiones() repository.PortalSesionRepository  { return r.sesiones }

type stubCuentaRepo struct {
	repository.PortalCuentaRepository
	byID   map[uuid.UUID]*models.PortalCuenta
	byCURP map[string]*models.PortalCuenta
}

func (r *stubCuentaRepo) Create(_ context.Context, c *models.PortalCuenta) error {
	cp := *c
	r.byID[c.ID] = &cp
	r.byCURP[c.CURPNormalizada] = &cp
	return nil
}
func (r *stubCuentaRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PortalCuenta, error) {
	c, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
func (r *stubCuentaRepo) GetByCURPNormalizada(_ context.Context, curp string) (*models.PortalCuenta, error) {
	c, ok := r.byCURP[curp]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
func (r *stubCuentaRepo) FindByNombre(_ context.Context, _ string) (*models.PortalCuenta, error) {
	return nil, repository.ErrNotFound
}

type stubDesafioRepo struct {
	repository.PortalDesafioRepository
	items map[uuid.UUID]*models.PortalDesafioOTP
}

func (r *stubDesafioRepo) Create(_ context.Context, d *models.PortalDesafioOTP) error {
	cp := *d
	r.items[d.ID] = &cp
	return nil
}
func (r *stubDesafioRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PortalDesafioOTP, error) {
	d, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *d
	return &cp, nil
}
func (r *stubDesafioRepo) Save(_ context.Context, d *models.PortalDesafioOTP) error {
	cp := *d
	r.items[d.ID] = &cp
	return nil
}

type stubSesionRepo struct {
	repository.PortalSesionRepository
}

func (r *stubSesionRepo) Create(_ context.Context, _ *models.PortalSesion) error { return nil }

func newPortalTestFixture(devMode bool) (*service, []string, *[]string) {
	cuentaID := uuid.New()
	curp := "ABCD000101HDFXXX01"
	cuenta := &models.PortalCuenta{
		ID:              cuentaID,
		Alias:           "Usuaria-123456",
		Nombre:          "Ana Perez",
		CURPNormalizada: curp,
		Telefono:        "5512345678",
		AceptaAviso:     true,
	}
	repos := &stubRepos{
		cuentas: &stubCuentaRepo{
			byID:   map[uuid.UUID]*models.PortalCuenta{cuentaID: cuenta},
			byCURP: map[string]*models.PortalCuenta{curp: cuenta},
		},
		desafios: &stubDesafioRepo{items: map[uuid.UUID]*models.PortalDesafioOTP{}},
		sesiones: &stubSesionRepo{},
	}
	logs := []string{}
	svc := New(&stubUOW{repos: repos}, devMode).(*service)
	svc.now = func() time.Time { return time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC) }
	svc.logf = func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	return svc, []string{curp}, &logs
}

func TestPortalOTPLoggingRespectsDevMode(t *testing.T) {
	codeRe := regexp.MustCompile(`code=(\d{6})\b`)

	t.Run("logs working 6-digit OTP on Login and CreateAccount when devMode=true", func(t *testing.T) {
		svc, keys, logs := newPortalTestFixture(true)

		ch, err := svc.Login(context.Background(), keys[0])
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

		*logs = nil
		ch2, err := svc.CreateAccount(context.Background(), pmodels.CreateAccountRequest{
			Nombre:      "Maria Lopez",
			CURP:        "MALO900101MDFXXX02",
			Telefono:    "5598765432",
			AceptaAviso: true,
		})
		if err != nil {
			t.Fatalf("CreateAccount failed: %v", err)
		}
		if len(*logs) != 1 {
			t.Fatalf("expected 1 log entry on CreateAccount, got %d", len(*logs))
		}
		m2 := codeRe.FindStringSubmatch((*logs)[0])
		if len(m2) != 2 {
			t.Fatalf("expected 6-digit code in log %q", (*logs)[0])
		}
		if _, err := svc.Verify(context.Background(), ch2.ChallengeID, m2[1]); err != nil {
			t.Fatalf("Verify with CreateAccount logged OTP failed: %v", err)
		}
	})

	t.Run("never logs OTP when devMode=false", func(t *testing.T) {
		svc, keys, logs := newPortalTestFixture(false)

		if _, err := svc.Login(context.Background(), keys[0]); err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if _, err := svc.CreateAccount(context.Background(), pmodels.CreateAccountRequest{
			Nombre:      "Maria Lopez",
			CURP:        "MALO900101MDFXXX02",
			Telefono:    "5598765432",
			AceptaAviso: true,
		}); err != nil {
			t.Fatalf("CreateAccount failed: %v", err)
		}
		if len(*logs) != 0 {
			t.Fatalf("expected 0 logs when devMode=false, got %v", *logs)
		}
	})
}
