package caseaccessrequest

import (
	"backend-hackathon/internal/httpx"
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	carmodels "backend-hackathon/internal/models/caseaccessrequest"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

type fakeUOW struct{ r *fakeRepos }

func (f *fakeUOW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) Repositories() repository.Repositories { return f.r }

type fakeRepos struct {
	repository.Repositories
	tokens *fakeTokens
	sols   *fakeSols
	claves *fakeClaves
	audit  *fakeAudit
}

func (r *fakeRepos) TokensVinculacion() repository.TokenVinculacionRepository          { return r.tokens }
func (r *fakeRepos) SolicitudesVinculacion() repository.SolicitudVinculacionRepository { return r.sols }
func (r *fakeRepos) ClavesRegistro() repository.ClaveRegistroRepository                { return r.claves }
func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository            { return r.audit }

type fakeTokens struct {
	repository.TokenVinculacionRepository
	byHash map[string]*models.TokenVinculacion
}

func (f *fakeTokens) GetByTokenHash(_ context.Context, h []byte) (*models.TokenVinculacion, error) {
	if t, ok := f.byHash[string(h)]; ok {
		return t, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeTokens) Consumir(_ context.Context, id uuid.UUID, at time.Time) (bool, error) {
	for _, t := range f.byHash {
		if t.ID == id && t.ConsumidoEn == nil && t.VenceEn.After(at) {
			t.ConsumidoEn = &at
			return true, nil
		}
	}
	return false, nil
}

type fakeSols struct {
	repository.SolicitudVinculacionRepository
	created []models.SolicitudVinculacion
}

func (f *fakeSols) Create(_ context.Context, s *models.SolicitudVinculacion) error {
	f.created = append(f.created, *s)
	return nil
}

func (f *fakeSols) List(_ context.Context, flt repository.SolicitudVinculacionFilter, _ repository.PageRequest) (repository.Page[models.SolicitudVinculacion], error) {
	var out []models.SolicitudVinculacion
	for _, s := range f.created {
		if (flt.Estado == nil || s.Estado == *flt.Estado) && (flt.SolicitanteID == nil || s.SolicitanteID == *flt.SolicitanteID) {
			out = append(out, s)
		}
	}
	return repository.Page[models.SolicitudVinculacion]{Items: out}, nil
}

type fakeClaves struct {
	repository.ClaveRegistroRepository
	m map[uuid.UUID]*models.ClaveRegistro
}

func (f *fakeClaves) Create(_ context.Context, c *models.ClaveRegistro) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	cp := *c
	f.m[c.ID] = &cp
	return nil
}

func (f *fakeClaves) GetByID(_ context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	if c, ok := f.m[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}

type fakeAudit struct {
	repository.EventoAuditoriaRepository
	acciones []string
}

func (f *fakeAudit) Append(_ context.Context, e *models.EventoAuditoria) error {
	f.acciones = append(f.acciones, e.Accion)
	return nil
}

func setup(t *testing.T) (*service, *fakeRepos, repository.ActorContext) {
	t.Helper()
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	r := &fakeRepos{
		tokens: &fakeTokens{byHash: map[string]*models.TokenVinculacion{}},
		sols:   &fakeSols{},
		claves: &fakeClaves{m: map[uuid.UUID]*models.ClaveRegistro{}},
		audit:  &fakeAudit{},
	}
	actor := repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	return &service{uow: &fakeUOW{r}, keyring: kr, secret: "s", now: time.Now}, r, actor
}

func addToken(r *fakeRepos, solicitante uuid.UUID, vence time.Time) (string, *models.TokenVinculacion) {
	token, hash, _ := security.NewOpaqueToken()
	t := &models.TokenVinculacion{ID: uuid.New(), ExpedienteID: uuid.New(), SolicitanteID: solicitante, TokenHash: hash, VenceEn: vence}
	r.tokens.byHash[string(hash)] = t
	return token, t
}

const motivo = "Atención psicológica solicitada por la persona"

func TestCreateConsumesTokenAndSealsMotivo(t *testing.T) {
	svc, r, actor := setup(t)
	token, tok := addToken(r, actor.ActorID, time.Now().Add(10*time.Minute))

	res, err := svc.Create(context.Background(), actor, carmodels.CreateRequest{LinkToken: token, Motivo: motivo})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Estado != models.SolicitudPendiente || len(r.sols.created) != 1 {
		t.Fatalf("respuesta %+v, creadas %d", res, len(r.sols.created))
	}
	s := r.sols.created[0]
	if s.ID != res.RequestID || s.ExpedienteID != tok.ExpedienteID || s.TokenID != tok.ID || s.SolicitanteID != actor.ActorID {
		t.Fatalf("solicitud mal ligada: %+v", s)
	}
	plain, err := svc.keyring.Open(context.Background(), r.claves, s.ClaveID, s.MotivoCifrado, MotivoAAD(s.ID))
	if err != nil || string(plain) != motivo {
		t.Fatalf("motivo = %q, %v", plain, err)
	}
	if tok.ConsumidoEn == nil {
		t.Fatal("el token debía quedar consumido")
	}
	if len(r.audit.acciones) != 1 || r.audit.acciones[0] != "case_access_request.create" {
		t.Fatalf("auditoría: %v", r.audit.acciones)
	}

	// Ya hay una solicitud pendiente para ese expediente: 409 request_exists.
	if _, err := svc.Create(context.Background(), actor, carmodels.CreateRequest{LinkToken: "lt_" + token, Motivo: motivo}); !errors.Is(err, ErrRequestExists) {
		t.Fatalf("pendiente repetida: err = %v, quiero ErrRequestExists", err)
	}
}

func TestCreateRejects(t *testing.T) {
	svc, r, actor := setup(t)
	vencido, _ := addToken(r, actor.ActorID, time.Now().Add(-time.Minute))
	ajeno, _ := addToken(r, uuid.New(), time.Now().Add(10*time.Minute))
	propio, _ := addToken(r, actor.ActorID, time.Now().Add(10*time.Minute))

	var ve *httpx.Error
	cases := []struct {
		name string
		req  carmodels.CreateRequest
		ok   func(error) bool
	}{
		{"token inexistente", carmodels.CreateRequest{LinkToken: "nope", Motivo: motivo}, func(e error) bool { return errors.Is(e, ErrLinkInvalid) }},
		{"token de otra persona", carmodels.CreateRequest{LinkToken: ajeno, Motivo: motivo}, func(e error) bool { return errors.Is(e, ErrLinkInvalid) }},
		{"token vencido", carmodels.CreateRequest{LinkToken: vencido, Motivo: motivo}, func(e error) bool { return errors.Is(e, ErrLinkExpired) }},
		{"motivo corto", carmodels.CreateRequest{LinkToken: propio, Motivo: "corto"}, func(e error) bool { return errors.As(e, &ve) }},
		{"sin token", carmodels.CreateRequest{Motivo: motivo}, func(e error) bool { return errors.As(e, &ve) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), actor, tc.req); !tc.ok(err) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	if len(r.sols.created) != 0 {
		t.Fatalf("no debía crearse ninguna solicitud: %d", len(r.sols.created))
	}
}

func TestCreateWithoutKeyring(t *testing.T) {
	svc, _, actor := setup(t)
	svc.keyring = nil
	if _, err := svc.Create(context.Background(), actor, carmodels.CreateRequest{LinkToken: "x", Motivo: motivo}); !errors.Is(err, ErrCryptoUnavailable) {
		t.Fatalf("err = %v", err)
	}
}
