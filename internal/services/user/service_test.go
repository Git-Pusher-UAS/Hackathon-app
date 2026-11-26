package user

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	usermodels "backend-hackathon/internal/models/user"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

type fakeUOW struct {
	repos *fakeRepos
}

func (f *fakeUOW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.repos)
}

func (f *fakeUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.repos)
}

func (f *fakeUOW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.repos)
}

func (f *fakeUOW) Repositories() repository.Repositories {
	return f.repos
}

type fakeRepos struct {
	repository.Repositories
	actores       *fakeActorRepo
	departamentos *fakeDepartamentoRepo
	personal      *fakePersonalRepo
	roles         *fakeRolRepo
	permisos      *fakePermisoRepo
	sesiones      *fakeSesionRepo
	mfa           *fakeMFARepo
	recuperacion  *fakeRecuperacionMFARepo
	operaciones   *fakeOperacionRepo
	claves        *fakeClaveRegistroRepo
	audit         *fakeAuditRepo
}

func (r *fakeRepos) Actores() repository.ActorRepository                       { return r.actores }
func (r *fakeRepos) Departamentos() repository.DepartamentoRepository        { return r.departamentos }
func (r *fakeRepos) Personal() repository.PersonalRepository                 { return r.personal }
func (r *fakeRepos) Roles() repository.RolRepository                           { return r.roles }
func (r *fakeRepos) Permisos() repository.PermisoRepository                    { return r.permisos }
func (r *fakeRepos) Sesiones() repository.SesionPersonalRepository             { return r.sesiones }
func (r *fakeRepos) MFA() repository.MFAPersonalRepository                     { return r.mfa }
func (r *fakeRepos) RecuperacionesMFA() repository.RecuperacionMFARepository { return r.recuperacion }
func (r *fakeRepos) Operaciones() repository.OperacionRepository               { return r.operaciones }
func (r *fakeRepos) ClavesRegistro() repository.ClaveRegistroRepository        { return r.claves }
func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository    { return r.audit }

type fakeActorRepo struct {
	repository.ActorRepository
	items     map[uuid.UUID]*models.Actor
	setActivo map[uuid.UUID]bool
}

func (r *fakeActorRepo) Create(_ context.Context, actor *models.Actor) error {
	if actor.ID == uuid.Nil {
		actor.ID = uuid.New()
	}
	cp := *actor
	r.items[actor.ID] = &cp
	return nil
}

func (r *fakeActorRepo) SetActivo(_ context.Context, id uuid.UUID, activo bool) error {
	if r.setActivo == nil {
		r.setActivo = make(map[uuid.UUID]bool)
	}
	r.setActivo[id] = activo
	if a, ok := r.items[id]; ok {
		a.Activo = activo
	}
	return nil
}

type fakeDepartamentoRepo struct {
	repository.DepartamentoRepository
	items map[uuid.UUID]*models.Departamento
}

func (r *fakeDepartamentoRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Departamento, error) {
	d, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

type fakePersonalRepo struct {
	repository.PersonalRepository
	items  map[uuid.UUID]*models.Personal
	listFn func(f repository.PersonalFilter, page repository.PageRequest) (repository.Page[models.Personal], error)
}

func (r *fakePersonalRepo) Create(_ context.Context, p *models.Personal) error {
	for _, existing := range r.items {
		if strings.EqualFold(existing.Usuario, p.Usuario) {
			return repository.ErrConflict
		}
	}
	cp := *p
	r.items[p.ID] = &cp
	return nil
}

func (r *fakePersonalRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	p, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakePersonalRepo) GetByUsuario(_ context.Context, usuario string) (*models.Personal, error) {
	for _, p := range r.items {
		if strings.EqualFold(p.Usuario, usuario) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakePersonalRepo) List(_ context.Context, f repository.PersonalFilter, page repository.PageRequest) (repository.Page[models.Personal], error) {
	if r.listFn != nil {
		return r.listFn(f, page)
	}
	var out []models.Personal
	for _, p := range r.items {
		out = append(out, *p)
	}
	return repository.Page[models.Personal]{Items: out}, nil
}

func (r *fakePersonalRepo) Update(_ context.Context, p *models.Personal, expectedVersion int64) error {
	existing, ok := r.items[p.ID]
	if !ok || existing.Version != expectedVersion {
		return repository.ErrVersionMismatch
	}
	cp := *p
	cp.Version = expectedVersion + 1
	r.items[p.ID] = &cp
	return nil
}

func (r *fakePersonalRepo) LockActivosConPermiso(_ context.Context, _ string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for id, p := range r.items {
		if p.Estado == models.PersonalActivo {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type fakeRolRepo struct {
	repository.RolRepository
	items map[uuid.UUID]*models.Rol
}

func (r *fakeRolRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Rol, error) {
	rol, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *rol
	return &cp, nil
}

func (r *fakeRolRepo) GetByCodigo(_ context.Context, codigo string) (*models.Rol, error) {
	for _, rol := range r.items {
		if rol.Codigo == codigo {
			cp := *rol
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeRolRepo) List(_ context.Context, soloActivos bool) ([]models.Rol, error) {
	var out []models.Rol
	for _, rol := range r.items {
		if soloActivos && !rol.Activo {
			continue
		}
		out = append(out, *rol)
	}
	return out, nil
}

type fakePermisoRepo struct {
	repository.PermisoRepository
	byPersonal map[uuid.UUID][]string
	byRol      map[uuid.UUID][]models.Permiso
}

func (r *fakePermisoRepo) CodigosEfectivos(_ context.Context, personalID uuid.UUID, _ time.Time) ([]string, error) {
	return r.byPersonal[personalID], nil
}

func (r *fakePermisoRepo) ListByRol(_ context.Context, rolID uuid.UUID) ([]models.Permiso, error) {
	return r.byRol[rolID], nil
}

type fakeMFARepo struct {
	repository.MFAPersonalRepository
	calls []string
}

func (r *fakeMFARepo) Invalidar(_ context.Context, _ uuid.UUID) error {
	r.calls = append(r.calls, "Invalidar")
	return nil
}

func (r *fakeMFARepo) DeletePendiente(_ context.Context, _ uuid.UUID) error {
	r.calls = append(r.calls, "DeletePendiente")
	return nil
}

type fakeRecuperacionMFARepo struct {
	repository.RecuperacionMFARepository
	items []*models.RecuperacionMFA
}

func (r *fakeRecuperacionMFARepo) Create(_ context.Context, rec *models.RecuperacionMFA) error {
	cp := *rec
	r.items = append(r.items, &cp)
	return nil
}

type fakeOperacionRepo struct {
	repository.OperacionRepository
	items []*models.Operacion
}

func (r *fakeOperacionRepo) Create(_ context.Context, op *models.Operacion) error {
	cp := *op
	r.items = append(r.items, &cp)
	return nil
}

type fakeClaveRegistroRepo struct {
	repository.ClaveRegistroRepository
	items map[uuid.UUID]*models.ClaveRegistro
}

func (r *fakeClaveRegistroRepo) Create(_ context.Context, c *models.ClaveRegistro) error {
	cp := *c
	r.items[c.ID] = &cp
	return nil
}

func (r *fakeClaveRegistroRepo) GetByID(_ context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	c, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

type fakeSesionRepo struct {
	repository.SesionPersonalRepository
	items     map[uuid.UUID]*models.SesionPersonal
	getErr    error
	revocarAt map[uuid.UUID]time.Time
	listFn    func(titularID uuid.UUID, soloActivas bool, now time.Time, page repository.PageRequest) (repository.Page[models.SesionPersonal], error)
}

func (r *fakeSesionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.SesionPersonal, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	s, ok := r.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSesionRepo) ListByTitular(_ context.Context, titularID uuid.UUID, soloActivas bool, now time.Time, page repository.PageRequest) (repository.Page[models.SesionPersonal], error) {
	if r.listFn != nil {
		return r.listFn(titularID, soloActivas, now, page)
	}
	return repository.Page[models.SesionPersonal]{}, nil
}

func (r *fakeSesionRepo) Revocar(_ context.Context, id uuid.UUID, at time.Time) error {
	s, ok := r.items[id]
	if !ok || s.RevocadoEn != nil {
		return repository.ErrNotFound
	}
	s.RevocadoEn = &at
	if r.revocarAt != nil {
		r.revocarAt[id] = at
	}
	return nil
}

func (r *fakeSesionRepo) RevocarTodasDeTitular(_ context.Context, titularID uuid.UUID, at time.Time) (int64, error) {
	var count int64
	for _, s := range r.items {
		if s.TitularID == titularID && s.RevocadoEn == nil {
			t := at
			s.RevocadoEn = &t
			count++
		}
	}
	return count, nil
}

type fakeAuditRepo struct {
	repository.EventoAuditoriaRepository
	events []*models.EventoAuditoria
}

func (r *fakeAuditRepo) Append(_ context.Context, e *models.EventoAuditoria) error {
	cp := *e
	r.events = append(r.events, &cp)
	return nil
}

func newTestFixture(now time.Time) (*service, *fakeRepos, uuid.UUID, uuid.UUID) {
	actorID := uuid.New()
	rolID := uuid.New()
	deptID := uuid.New()
	sessionID := uuid.New()

	repos := &fakeRepos{
		actores: &fakeActorRepo{
			items: map[uuid.UUID]*models.Actor{
				actorID: {ID: actorID, Tipo: models.ActorPersonal, Activo: true},
			},
			setActivo: make(map[uuid.UUID]bool),
		},
		departamentos: &fakeDepartamentoRepo{
			items: map[uuid.UUID]*models.Departamento{
				deptID: {ID: deptID, Codigo: "TS", Nombre: "Trabajo Social", Activo: true},
			},
		},
		personal: &fakePersonalRepo{
			items: map[uuid.UUID]*models.Personal{
				actorID: {
					ID:              actorID,
					Usuario:         "ana.lopez",
					NombreLaboral:   "Ana López",
					RolID:           rolID,
					DepartamentoID:  deptID,
					Estado:          models.PersonalActivo,
					VersionPermisos: 1,
					Version:         1,
				},
			},
		},
		roles: &fakeRolRepo{
			items: map[uuid.UUID]*models.Rol{
				rolID: {ID: rolID, Codigo: "TrabajoSocial", Nombre: "Trabajo Social", Activo: true},
			},
		},
		permisos: &fakePermisoRepo{
			byPersonal: map[uuid.UUID][]string{
				actorID: {"case_files.read", "notes.create"},
			},
			byRol: make(map[uuid.UUID][]models.Permiso),
		},
		sesiones: &fakeSesionRepo{
			items: map[uuid.UUID]*models.SesionPersonal{
				sessionID: {
					ID:              sessionID,
					TitularID:       actorID,
					UltimaActividad: now.Add(-2 * time.Minute),
					VenceAbsoluto:   now.Add(10 * time.Hour),
					Dispositivo:     "Mozilla/5.0",
					CreadoEn:        now.Add(-1 * time.Hour),
				},
			},
			revocarAt: make(map[uuid.UUID]time.Time),
		},
		mfa:          &fakeMFARepo{},
		recuperacion: &fakeRecuperacionMFARepo{},
		operaciones:  &fakeOperacionRepo{},
		claves:       &fakeClaveRegistroRepo{items: make(map[uuid.UUID]*models.ClaveRegistro)},
		audit:        &fakeAuditRepo{},
	}

	svc := New(&fakeUOW{repos: repos}, nil, "test-secret")
	svc.now = func() time.Time { return now }
	return svc, repos, actorID, sessionID
}

func TestMeSuccessAndEmptyPermissions(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, actorID, sessionID := newTestFixture(now)

	res, err := svc.Me(context.Background(), actorID, sessionID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if res.Actor.ID != actorID || res.Actor.Rol != "TrabajoSocial" || res.Actor.DisplayName != "Ana López" {
		t.Fatalf("actor inesperado: %+v", res.Actor)
	}
	if len(res.Actor.Permissions) != 2 {
		t.Fatalf("permissions = %v, esperado 2", res.Actor.Permissions)
	}
	wantIdle := now.Add(-2 * time.Minute).Add(auth.IdleTimeout)
	if !res.IdleExpiresAt.Equal(wantIdle) {
		t.Fatalf("IdleExpiresAt = %v, esperado %v", res.IdleExpiresAt, wantIdle)
	}

	// Nil permissions from repository should serialize as empty slice, not nil.
	repos.permisos.byPersonal[actorID] = nil
	res, err = svc.Me(context.Background(), actorID, sessionID)
	if err != nil {
		t.Fatalf("Me sin permisos: %v", err)
	}
	if res.Actor.Permissions == nil || len(res.Actor.Permissions) != 0 {
		t.Fatalf("permissions = %#v, esperado []string{}", res.Actor.Permissions)
	}
}

func TestMeRejectsInvalidStates(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	t.Run("usuario inexistente", func(t *testing.T) {
		svc, _, _, sessionID := newTestFixture(now)
		_, err := svc.Me(context.Background(), uuid.New(), sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("usuario suspendido", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		repos.personal.items[actorID].Estado = models.PersonalSuspendido
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("usuario bloqueado", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		until := now.Add(5 * time.Minute)
		repos.personal.items[actorID].BloqueadoHasta = &until
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("sesión revocada", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		rev := now.Add(-1 * time.Minute)
		repos.sesiones.items[sessionID].RevocadoEn = &rev
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("sesión vencida absoluto", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		repos.sesiones.items[sessionID].VenceAbsoluto = now
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("sesión vencida por inactividad", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		repos.sesiones.items[sessionID].UltimaActividad = now.Add(-auth.IdleTimeout)
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})

	t.Run("sesión de otro titular", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		repos.sesiones.items[sessionID].TitularID = uuid.New()
		_, err := svc.Me(context.Background(), actorID, sessionID)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, esperado ErrUnauthenticated", err)
		}
	})
}

func TestListMySessions(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, actorID, currentID := newTestFixture(now)
	otherID := uuid.New()

	var capturedSoloActivas bool
	var capturedPlainCursor string
	repos.sesiones.listFn = func(titularID uuid.UUID, soloActivas bool, _ time.Time, page repository.PageRequest) (repository.Page[models.SesionPersonal], error) {
		if titularID != actorID {
			t.Fatalf("titularID = %v, esperado %v", titularID, actorID)
		}
		capturedSoloActivas = soloActivas
		capturedPlainCursor = page.Cursor
		return repository.Page[models.SesionPersonal]{
			Items: []models.SesionPersonal{
				{
					ID:              currentID,
					TitularID:       actorID,
					CreadoEn:        now.Add(-20 * time.Minute),
					UltimaActividad: now.Add(-1 * time.Minute),
					VenceAbsoluto:   now.Add(8 * time.Hour),
					Dispositivo:     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)",
				},
				{
					ID:              otherID,
					TitularID:       actorID,
					CreadoEn:        now.Add(-40 * time.Minute),
					UltimaActividad: now.Add(-5 * time.Minute),
					VenceAbsoluto:   now.Add(7 * time.Hour),
					Dispositivo:     "Mozilla/5.0 (X11; Linux x86_64)",
				},
			},
			NextCursor: "raw-cursor-1",
		}, nil
	}

	inCursor := httpx.EncodeCursor(svc.secret, actorID, cursorScopeSessions, httpx.FilterHash(), "page-0")
	page, err := svc.ListMySessions(context.Background(), actorID, currentID, repository.PageRequest{
		Limit:  10,
		Cursor: inCursor,
	})
	if err != nil {
		t.Fatalf("ListMySessions: %v", err)
	}
	if !capturedSoloActivas {
		t.Fatal("esperado soloActivas=true")
	}
	if capturedPlainCursor != "page-0" {
		t.Fatalf("plainCursor = %q, esperado page-0", capturedPlainCursor)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, esperado 2", len(page.Items))
	}
	if !page.Items[0].Current || page.Items[0].DeviceLabel != "Dispositivo móvil" {
		t.Fatalf("item[0] inesperado: %+v", page.Items[0])
	}
	if page.Items[1].Current || page.Items[1].DeviceLabel != "Navegador web" {
		t.Fatalf("item[1] inesperado: %+v", page.Items[1])
	}
	if page.NextCursor == nil {
		t.Fatal("NextCursor nil, esperado cursor firmado")
	}
	plainNext, err := httpx.DecodeCursor(svc.secret, *page.NextCursor, actorID, cursorScopeSessions, httpx.FilterHash())
	if err != nil || plainNext != "raw-cursor-1" {
		t.Fatalf("NextCursor decodificado = (%q, %v), esperado raw-cursor-1", plainNext, err)
	}
}

func TestListMySessionsRejectsInvalidCursor(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, _, actorID, currentID := newTestFixture(now)

	foreignCursor := httpx.EncodeCursor(svc.secret, uuid.New(), cursorScopeSessions, httpx.FilterHash(), "p")
	_, err := svc.ListMySessions(context.Background(), actorID, currentID, repository.PageRequest{
		Limit:  10,
		Cursor: foreignCursor,
	})
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("err = %v, esperado ValidationError", err)
	}
}

func TestRevokeMySession(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	t.Run("revoca sesión propia y audita éxito", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

		if err := svc.RevokeMySession(context.Background(), actor, actorID, sessionID); err != nil {
			t.Fatalf("RevokeMySession: %v", err)
		}
		if repos.sesiones.items[sessionID].RevocadoEn == nil {
			t.Fatal("sesión no fue marcada como revocada")
		}
		if len(repos.audit.events) != 1 {
			t.Fatalf("eventos = %d, esperado 1", len(repos.audit.events))
		}
		ev := repos.audit.events[0]
		if ev.Accion != "users.session.revoke" || ev.TipoRecurso != "sesion" || ev.Resultado != models.ResultadoExito {
			t.Fatalf("evento inesperado: %+v", ev)
		}
	})

	t.Run("idempotente si ya estaba revocada", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		prev := now.Add(-5 * time.Minute)
		repos.sesiones.items[sessionID].RevocadoEn = &prev
		actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

		if err := svc.RevokeMySession(context.Background(), actor, actorID, sessionID); err != nil {
			t.Fatalf("RevokeMySession idempotente: %v", err)
		}
		if !repos.sesiones.items[sessionID].RevocadoEn.Equal(prev) {
			t.Fatal("RevocadoEn original fue sobreescrito")
		}
	})

	t.Run("sesión ajena devuelve ErrNotFound y audita denegación", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		repos.sesiones.items[sessionID].TitularID = uuid.New()
		actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

		err := svc.RevokeMySession(context.Background(), actor, actorID, sessionID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, esperado ErrNotFound", err)
		}
		if len(repos.audit.events) != 1 || repos.audit.events[0].Resultado != models.ResultadoDenegado || repos.audit.events[0].TipoRecurso != "sesion" {
			t.Fatalf("auditoría de denegación inesperada: %+v", repos.audit.events)
		}
	})

	t.Run("error inesperado de repositorio no se convierte en ErrNotFound", func(t *testing.T) {
		svc, repos, actorID, sessionID := newTestFixture(now)
		dbErr := errors.New("connection lost")
		repos.sesiones.getErr = dbErr
		actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

		err := svc.RevokeMySession(context.Background(), actor, actorID, sessionID)
		if !errors.Is(err, dbErr) {
			t.Fatalf("err = %v, esperado %v", err, dbErr)
		}
		if len(repos.audit.events) != 0 {
			t.Fatalf("no debía auditar denegación en error de BD, eventos = %d", len(repos.audit.events))
		}
	})
}

func TestListUsers(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, actorID, _ := newTestFixture(now)

	var gotSearch string
	repos.personal.listFn = func(f repository.PersonalFilter, _ repository.PageRequest) (repository.Page[models.Personal], error) {
		gotSearch = f.Busqueda
		return repository.Page[models.Personal]{
			Items:      []models.Personal{*repos.personal.items[actorID]},
			NextCursor: "next-u",
		}, nil
	}

	inCursor := httpx.EncodeCursor(svc.secret, actorID, cursorScopeUsers, httpx.FilterHash("ana"), "prev-u")
	page, err := svc.List(context.Background(), actorID, "  ana  ", repository.PageRequest{Limit: 10, Cursor: inCursor})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotSearch != "ana" {
		t.Fatalf("search = %q, esperado ana", gotSearch)
	}
	if len(page.Items) != 1 || page.Items[0].Rol != "TrabajoSocial" || page.Items[0].VersionTag == "" {
		t.Fatalf("items inesperados: %+v", page.Items)
	}
	if page.NextCursor == nil {
		t.Fatal("esperado NextCursor")
	}

	if _, err := svc.List(context.Background(), actorID, "otro", repository.PageRequest{Limit: 10, Cursor: inCursor}); err == nil {
		t.Fatal("esperado error por cursor con filtro distinto")
	}
}

func TestCreateUser(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, actorID, _ := newTestFixture(now)
	var deptID uuid.UUID
	for id := range repos.departamentos.items {
		deptID = id
	}
	actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

	t.Run("crea cuenta nueva, cifra contraseña y audita", func(t *testing.T) {
		res, err := svc.Create(context.Background(), actor, usermodels.CreateRequest{
			Usuario:        "  Carlos.Perez ",
			NombreLaboral:  "Carlos Pérez",
			Rol:            "TrabajoSocial",
			DepartamentoID: deptID.String(),
			Password:       "ClaveSegura1234",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if res.Usuario != "carlos.perez" || res.Rol != "TrabajoSocial" || res.Estado != models.PersonalActivo {
			t.Fatalf("summary inesperado: %+v", res)
		}
		v, err := httpx.DecodeVersionTag(svc.secret, res.VersionTag)
		if err != nil || v != 1 {
			t.Fatalf("versionTag = (%d, %v), esperado 1", v, err)
		}
		created := repos.personal.items[res.ID]
		if !created.RequiereCambioPassword || !strings.HasPrefix(created.PasswordHash, "$argon2id$") {
			t.Fatalf("personal creado inválido: %+v", created)
		}
		if len(repos.audit.events) == 0 || repos.audit.events[len(repos.audit.events)-1].Accion != "users.create" {
			t.Fatalf("auditoría inesperada: %+v", repos.audit.events)
		}
	})

	t.Run("rechaza usuario duplicado", func(t *testing.T) {
		_, err := svc.Create(context.Background(), actor, usermodels.CreateRequest{
			Usuario:        "ANA.LOPEZ",
			NombreLaboral:  "Duplicada",
			Rol:            "TrabajoSocial",
			DepartamentoID: deptID.String(),
			Password:       "ClaveSegura1234",
		})
		if !errors.Is(err, ErrDuplicateUsername) {
			t.Fatalf("err = %v, esperado ErrDuplicateUsername", err)
		}
	})

	t.Run("valida campos de entrada", func(t *testing.T) {
		_, err := svc.Create(context.Background(), actor, usermodels.CreateRequest{
			Usuario:        "con espacio",
			NombreLaboral:  "Nombre",
			Rol:            "TrabajoSocial",
			DepartamentoID: deptID.String(),
			Password:       "ClaveSegura1234",
		})
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("err = %v, esperado ValidationError", err)
		}
	})
}

func TestUpdateUser(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, adminID, _ := newTestFixture(now)

	adminRoleID := uuid.New()
	var tsRoleID, deptID uuid.UUID
	for id := range repos.roles.items {
		tsRoleID = id
	}
	for id := range repos.departamentos.items {
		deptID = id
	}
	repos.roles.items[adminRoleID] = &models.Rol{ID: adminRoleID, Codigo: "AdministradorTI", Nombre: "Admin TI", Activo: true}
	repos.permisos.byRol[adminRoleID] = []models.Permiso{{Codigo: "admin"}}
	repos.personal.items[adminID].RolID = adminRoleID

	targetID := uuid.New()
	targetSessID := uuid.New()
	repos.actores.items[targetID] = &models.Actor{ID: targetID, Tipo: models.ActorPersonal, Activo: true}
	repos.personal.items[targetID] = &models.Personal{
		ID:             targetID,
		Usuario:        "juan.diaz",
		NombreLaboral:  "Juan Díaz",
		RolID:          tsRoleID,
		DepartamentoID: deptID,
		Estado:         models.PersonalActivo,
		Version:        3,
	}
	repos.sesiones.items[targetSessID] = &models.SesionPersonal{ID: targetSessID, TitularID: targetID}

	actor := repository.ActorContext{ActorID: adminID, RequestID: uuid.New()}
	tag := httpx.EncodeVersionTag(svc.secret, 3)

	t.Run("rechaza autoelevación", func(t *testing.T) {
		suspended := models.PersonalSuspendido
		_, err := svc.Update(context.Background(), actor, adminID, usermodels.UpdateRequest{Estado: &suspended}, tag)
		if !errors.Is(err, ErrSelfElevation) {
			t.Fatalf("err = %v, esperado ErrSelfElevation", err)
		}
	})

	t.Run("suspende usuario, sincroniza actor y revoca sesiones", func(t *testing.T) {
		suspended := models.PersonalSuspendido
		res, err := svc.Update(context.Background(), actor, targetID, usermodels.UpdateRequest{Estado: &suspended}, tag)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if res.Estado != models.PersonalSuspendido {
			t.Fatalf("estado = %s, esperado suspendido", res.Estado)
		}
		if v, err := httpx.DecodeVersionTag(svc.secret, res.VersionTag); err != nil || v != 4 {
			t.Fatalf("versionTag = (%d, %v), esperado 4", v, err)
		}
		if active, ok := repos.actores.setActivo[targetID]; !ok || active {
			t.Fatalf("actores.activo no sincronizado a false: %v (%v)", active, ok)
		}
		if repos.sesiones.items[targetSessID].RevocadoEn == nil {
			t.Fatal("sesión del usuario actualizado no fue revocada")
		}
	})

	t.Run("impide suspender al último administrador activo", func(t *testing.T) {
		otherAdminCaller := repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
		adminTag := httpx.EncodeVersionTag(svc.secret, 1)
		suspended := models.PersonalSuspendido
		_, err := svc.Update(context.Background(), otherAdminCaller, adminID, usermodels.UpdateRequest{Estado: &suspended}, adminTag)
		if !errors.Is(err, ErrLastAdmin) {
			t.Fatalf("err = %v, esperado ErrLastAdmin", err)
		}
	})

	t.Run("falla con 412 si If-Match es obsoleto", func(t *testing.T) {
		newRole := "AdministradorTI"
		_, err := svc.Update(context.Background(), actor, targetID, usermodels.UpdateRequest{Rol: &newRole}, tag)
		if !errors.Is(err, ErrVersionMismatch) {
			t.Fatalf("err = %v, esperado ErrVersionMismatch", err)
		}
	})
}

func TestRevokeAllSessionsAndResetMFA(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc, repos, actorID, sessionID := newTestFixture(now)
	actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

	t.Run("503 si no hay keyring", func(t *testing.T) {
		if err := svc.RevokeAllSessions(context.Background(), actor, actorID, "motivo suficientemente largo"); !errors.Is(err, ErrCryptoUnavailable) {
			t.Fatalf("RevokeAllSessions err = %v, esperado ErrCryptoUnavailable", err)
		}
		if _, err := svc.ResetMFA(context.Background(), actor, actorID, "motivo suficientemente largo", true); !errors.Is(err, ErrCryptoUnavailable) {
			t.Fatalf("ResetMFA err = %v, esperado ErrCryptoUnavailable", err)
		}
	})

	keyring, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	svc.keyring = keyring

	t.Run("RevokeAllSessions sella justificación y revoca sesiones", func(t *testing.T) {
		if err := svc.RevokeAllSessions(context.Background(), actor, actorID, "  incidente de seguridad reportado  "); err != nil {
			t.Fatalf("RevokeAllSessions: %v", err)
		}
		if repos.sesiones.items[sessionID].RevocadoEn == nil {
			t.Fatal("sesión no revocada")
		}
		if len(repos.operaciones.items) != 1 {
			t.Fatalf("operaciones = %d, esperado 1", len(repos.operaciones.items))
		}
		op := repos.operaciones.items[0]
		plain, err := keyring.Open(context.Background(), repos.claves, op.ClaveID, op.ResultadoCifrado, []byte("seguridad.operaciones:resultado_cifrado:"+op.ID.String()))
		if err != nil {
			t.Fatalf("Open operación: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(plain, &payload); err != nil || payload["motivo"] != "incidente de seguridad reportado" {
			t.Fatalf("payload descifrado inesperado: %s (%v)", string(plain), err)
		}
	})

	t.Run("ResetMFA rechaza y audita cuando verificacionPresencial es false", func(t *testing.T) {
		beforeEvents := len(repos.audit.events)
		_, err := svc.ResetMFA(context.Background(), actor, actorID, "motivo suficientemente largo", false)
		if !errors.Is(err, ErrMFAVerificationRequired) {
			t.Fatalf("err = %v, esperado ErrMFAVerificationRequired", err)
		}
		if len(repos.audit.events) != beforeEvents+1 || repos.audit.events[beforeEvents].Resultado != models.ResultadoDenegado {
			t.Fatalf("esperada auditoría de denegación: %+v", repos.audit.events)
		}
	})

	t.Run("ResetMFA borra pendiente antes de invalidar y sella recuperación", func(t *testing.T) {
		repos.sesiones.items[sessionID].RevocadoEn = nil
		repos.mfa.calls = nil

		resetID, err := svc.ResetMFA(context.Background(), actor, actorID, "verificación presencial realizada", true)
		if err != nil {
			t.Fatalf("ResetMFA: %v", err)
		}
		if resetID == uuid.Nil {
			t.Fatal("resetID vacío")
		}
		if len(repos.mfa.calls) != 2 || repos.mfa.calls[0] != "DeletePendiente" || repos.mfa.calls[1] != "Invalidar" {
			t.Fatalf("orden MFA = %v, esperado [DeletePendiente Invalidar]", repos.mfa.calls)
		}
		if repos.sesiones.items[sessionID].RevocadoEn == nil {
			t.Fatal("sesión no revocada tras ResetMFA")
		}
		if len(repos.recuperacion.items) != 1 || repos.recuperacion.items[0].ID != resetID {
			t.Fatalf("recuperaciones = %+v", repos.recuperacion.items)
		}
	})
}
