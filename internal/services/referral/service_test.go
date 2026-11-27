package referral

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const secret = "test-secret"

// ---- fakes: solo los métodos que usa el servicio; el resto entra en pánico ----

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
	canal     *fakeCanal
	hist      *fakeHist
	personal  map[uuid.UUID]*models.Personal
	deps      map[uuid.UUID]*models.Departamento
	exps      map[uuid.UUID]*models.Expediente
	asig      *fakeAsig
	claves    *fakeClaves
	auditados []string
}

func (r *fakeRepos) Canalizaciones() repository.CanalizacionRepository { return r.canal }
func (r *fakeRepos) HistorialCanalizaciones() repository.HistorialCanalizacionRepository {
	return r.hist
}
func (r *fakeRepos) Personal() repository.PersonalRepository                { return fakePersonal{m: r.personal} }
func (r *fakeRepos) Departamentos() repository.DepartamentoRepository       { return fakeDeps{m: r.deps} }
func (r *fakeRepos) Expedientes() repository.ExpedienteRepository           { return fakeExps{m: r.exps} }
func (r *fakeRepos) Asignaciones() repository.AsignacionRepository          { return r.asig }
func (r *fakeRepos) ClavesRegistro() repository.ClaveRegistroRepository     { return r.claves }
func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository { return fakeAudit{r: r} }

type fakeCanal struct {
	repository.CanalizacionRepository
	items map[uuid.UUID]*models.CanalizacionInterna
}

func (f *fakeCanal) GetByIDForUpdate(_ context.Context, id uuid.UUID) (*models.CanalizacionInterna, error) {
	c, ok := f.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCanal) Update(_ context.Context, c *models.CanalizacionInterna, expected int64) error {
	cur := f.items[c.ID]
	if cur.Version != expected {
		return repository.ErrVersionMismatch
	}
	cp := *c
	cp.Version = expected + 1
	f.items[c.ID] = &cp
	return nil
}

type fakeHist struct {
	repository.HistorialCanalizacionRepository
	items []models.HistorialCanalizacion
}

func (f *fakeHist) Append(_ context.Context, h *models.HistorialCanalizacion) error {
	f.items = append(f.items, *h)
	return nil
}

type fakePersonal struct {
	repository.PersonalRepository
	m map[uuid.UUID]*models.Personal
}

func (f fakePersonal) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	if p, ok := f.m[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

type fakeDeps struct {
	repository.DepartamentoRepository
	m map[uuid.UUID]*models.Departamento
}

func (f fakeDeps) GetByID(_ context.Context, id uuid.UUID) (*models.Departamento, error) {
	if d, ok := f.m[id]; ok {
		return d, nil
	}
	return nil, repository.ErrNotFound
}

type fakeExps struct {
	repository.ExpedienteRepository
	m map[uuid.UUID]*models.Expediente
}

func (f fakeExps) GetByID(_ context.Context, id uuid.UUID) (*models.Expediente, error) {
	if e, ok := f.m[id]; ok {
		return e, nil
	}
	return nil, repository.ErrNotFound
}

type fakeAsig struct {
	repository.AsignacionRepository
	created []models.Asignacion
}

func (f *fakeAsig) FindVigente(_ context.Context, exp, per, dep uuid.UUID) (*models.Asignacion, error) {
	for i := range f.created {
		a := f.created[i]
		if a.ExpedienteID == exp && a.PersonalID == per && a.DepartamentoID == dep {
			return &a, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeAsig) Create(_ context.Context, a *models.Asignacion) error {
	f.created = append(f.created, *a)
	return nil
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
	r *fakeRepos
}

func (f fakeAudit) Append(_ context.Context, e *models.EventoAuditoria) error {
	f.r.auditados = append(f.r.auditados, e.Accion)
	return nil
}

// ---- escenario ----

type fixture struct {
	svc                       *service
	repos                     *fakeRepos
	refID                     uuid.UUID
	origenUser, destinoUser   repository.ActorContext
	ajeno                     repository.ActorContext
	origenDep, destinoDep, ex uuid.UUID
}

func newFixture(t *testing.T, estado models.EstadoCanalizacion) *fixture {
	t.Helper()
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{refID: uuid.New(), origenDep: uuid.New(), destinoDep: uuid.New(), ex: uuid.New()}
	otroDep := uuid.New()
	f.origenUser = repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	f.destinoUser = repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	f.ajeno = repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	f.repos = &fakeRepos{
		canal: &fakeCanal{items: map[uuid.UUID]*models.CanalizacionInterna{}},
		hist:  &fakeHist{},
		personal: map[uuid.UUID]*models.Personal{
			f.origenUser.ActorID:  {ID: f.origenUser.ActorID, DepartamentoID: f.origenDep},
			f.destinoUser.ActorID: {ID: f.destinoUser.ActorID, DepartamentoID: f.destinoDep},
			f.ajeno.ActorID:       {ID: f.ajeno.ActorID, DepartamentoID: otroDep},
		},
		deps: map[uuid.UUID]*models.Departamento{
			f.origenDep:  {ID: f.origenDep, Nombre: "Trabajo social"},
			f.destinoDep: {ID: f.destinoDep, Nombre: "Psicología"},
		},
		exps:   map[uuid.UUID]*models.Expediente{f.ex: {ID: f.ex, Alias: "Caso Bugambilia", Estado: models.ExpedienteAbierto}},
		asig:   &fakeAsig{},
		claves: &fakeClaves{m: map[uuid.UUID]*models.ClaveRegistro{}},
	}
	clave, resumen, err := kr.Seal(context.Background(), f.repos.claves, modulo, "canalizacion", f.refID,
		[]byte("Requiere acompañamiento"), aad("canalizacion", "resumen", f.refID))
	if err != nil {
		t.Fatal(err)
	}
	f.repos.canal.items[f.refID] = &models.CanalizacionInterna{ID: f.refID, ExpedienteID: f.ex, OrigenID: f.origenDep,
		DestinoID: f.destinoDep, ResumenCifrado: resumen, Estado: estado, ClaveID: clave, Version: 1}
	f.svc = &service{uow: &fakeUOW{f.repos}, keyring: kr, secret: secret, now: time.Now}
	return f
}

func (f *fixture) tag(v int64) string { return httpx.FormatETag(httpx.EncodeVersionTag(secret, v)) }

func ptr(s string) *string { return &s }

func TestTransitionAcceptAssignsDestino(t *testing.T) {
	f := newFixture(t, models.CanalizacionPendiente)
	res, err := f.svc.Transition(context.Background(), f.destinoUser, f.refID, f.tag(1), refmodels.TransitionRequest{Action: refmodels.ActionAccept})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if res.Estado != models.CanalizacionAceptada || res.Resumen != "Requiere acompañamiento" || res.Alias != "Caso Bugambilia" {
		t.Fatalf("vista inesperada: %+v", res)
	}
	if v, _ := httpx.DecodeVersionTag(secret, res.VersionTag); v != 2 {
		t.Fatalf("versionTag = v%d, quiero v2", v)
	}
	if len(f.repos.asig.created) != 1 || f.repos.asig.created[0].PersonalID != f.destinoUser.ActorID ||
		f.repos.asig.created[0].DepartamentoID != f.destinoDep {
		t.Fatalf("asignación: %+v", f.repos.asig.created)
	}
	if got := f.repos.canal.items[f.refID].AsignacionDestinoID; got == nil || *got != f.repos.asig.created[0].ID {
		t.Fatalf("asignacion_destino_id no quedó ligado")
	}
	if len(f.repos.hist.items) != 1 || *f.repos.hist.items[0].EstadoAnterior != models.CanalizacionPendiente {
		t.Fatalf("historial: %+v", f.repos.hist.items)
	}
	if len(f.repos.auditados) != 1 || f.repos.auditados[0] != "referral.accept" {
		t.Fatalf("auditoría: %v", f.repos.auditados)
	}
}

func TestTransitionRules(t *testing.T) {
	cases := []struct {
		name   string
		estado models.EstadoCanalizacion
		actor  func(*fixture) repository.ActorContext
		ifm    func(*fixture) string
		req    refmodels.TransitionRequest
		want   error
	}{
		{"sin If-Match", models.CanalizacionPendiente, dst, func(*fixture) string { return "" },
			refmodels.TransitionRequest{Action: "accept"}, ErrPreconditionRequired},
		{"versión vieja", models.CanalizacionPendiente, dst, func(f *fixture) string { return f.tag(7) },
			refmodels.TransitionRequest{Action: "accept"}, ErrVersionMismatch},
		{"tag con firma ajena", models.CanalizacionPendiente, dst, func(*fixture) string { return `"bogus"` },
			refmodels.TransitionRequest{Action: "accept"}, ErrVersionMismatch},
		{"departamento ajeno no la ve", models.CanalizacionPendiente, func(f *fixture) repository.ActorContext { return f.ajeno }, v1,
			refmodels.TransitionRequest{Action: "accept"}, ErrNotFound},
		{"origen no puede aceptar", models.CanalizacionPendiente, org, v1,
			refmodels.TransitionRequest{Action: "accept"}, ErrInvalidTransition},
		{"destino no puede cancelar", models.CanalizacionPendiente, dst, v1,
			refmodels.TransitionRequest{Action: "cancel", Motivo: ptr("ya no hace falta atención")}, ErrInvalidTransition},
		{"completar sin aceptar", models.CanalizacionPendiente, dst, v1,
			refmodels.TransitionRequest{Action: "complete"}, ErrInvalidTransition},
		{"aceptar dos veces", models.CanalizacionAceptada, dst, v1,
			refmodels.TransitionRequest{Action: "accept"}, ErrInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, tc.estado)
			_, err := f.svc.Transition(context.Background(), tc.actor(f), f.refID, tc.ifm(f), tc.req)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, quiero %v", err, tc.want)
			}
			if f.repos.canal.items[f.refID].Estado != tc.estado || len(f.repos.hist.items) != 0 {
				t.Fatalf("no debía cambiar nada")
			}
		})
	}
}

func TestTransitionMotivo(t *testing.T) {
	f := newFixture(t, models.CanalizacionPendiente)
	_, err := f.svc.Transition(context.Background(), f.destinoUser, f.refID, f.tag(1), refmodels.TransitionRequest{Action: "reject"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("reject sin motivo: err = %v", err)
	}
	if _, err := f.svc.Transition(context.Background(), f.destinoUser, f.refID, f.tag(1), refmodels.TransitionRequest{Action: "jump"}); !errors.As(err, &ve) {
		t.Fatalf("acción desconocida: err = %v", err)
	}

	res, err := f.svc.Transition(context.Background(), f.origenUser, f.refID, f.tag(1),
		refmodels.TransitionRequest{Action: "cancel", Motivo: ptr("la persona fue atendida en otra área")})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if res.Estado != models.CanalizacionCancelada {
		t.Fatalf("estado = %s", res.Estado)
	}
	h := f.repos.hist.items[0]
	if len(h.MotivoCifrado) == 0 {
		t.Fatal("el motivo debía quedar cifrado en el historial")
	}
	plain, err := f.svc.keyring.Open(context.Background(), f.repos.claves, h.ClaveID, h.MotivoCifrado,
		aad("historial_canalizacion", "motivo", h.ID))
	if err != nil || string(plain) != "la persona fue atendida en otra área" {
		t.Fatalf("motivo descifrado = %q, %v", plain, err)
	}
}

func TestTransitionCaseClosed(t *testing.T) {
	f := newFixture(t, models.CanalizacionPendiente)
	f.repos.exps[f.ex].Estado = models.ExpedienteCerrado
	_, err := f.svc.Transition(context.Background(), f.destinoUser, f.refID, f.tag(1), refmodels.TransitionRequest{Action: "accept"})
	if !errors.Is(err, ErrCaseClosed) {
		t.Fatalf("err = %v", err)
	}
}

func dst(f *fixture) repository.ActorContext { return f.destinoUser }
func org(f *fixture) repository.ActorContext { return f.origenUser }
func v1(f *fixture) string                   { return f.tag(1) }
