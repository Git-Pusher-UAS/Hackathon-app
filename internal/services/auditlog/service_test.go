package auditlog

import (
	"context"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

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
	eventos  []models.EventoAuditoria
	filtro   repository.AuditoriaFilter
	personal map[uuid.UUID]*models.Personal
	roles    map[uuid.UUID]*models.Rol
}

type fakeEventos struct {
	repository.EventoAuditoriaRepository
	r *fakeRepos
}

func (f fakeEventos) List(_ context.Context, flt repository.AuditoriaFilter, _ repository.PageRequest) (repository.Page[models.EventoAuditoria], error) {
	f.r.filtro = flt
	return repository.Page[models.EventoAuditoria]{Items: append([]models.EventoAuditoria(nil), f.r.eventos...)}, nil
}

func (f fakeEventos) Append(_ context.Context, e *models.EventoAuditoria) error {
	f.r.eventos = append(f.r.eventos, *e)
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

type fakeRoles struct {
	repository.RolRepository
	m map[uuid.UUID]*models.Rol
}

func (f fakeRoles) GetByID(_ context.Context, id uuid.UUID) (*models.Rol, error) { return f.m[id], nil }

func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository {
	return fakeEventos{r: r}
}
func (r *fakeRepos) Personal() repository.PersonalRepository { return fakePersonal{m: r.personal} }
func (r *fakeRepos) Roles() repository.RolRepository         { return fakeRoles{m: r.roles} }

func TestLogs(t *testing.T) {
	ana, rol, exp, canal := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	detalle := "Reveló teléfono. Motivo: Contacto de seguimiento"
	r := &fakeRepos{
		eventos: []models.EventoAuditoria{
			{ID: uuid.New(), ActorID: &ana, Accion: "casefile.disclose.telefono", TipoRecurso: "expediente", RecursoID: &exp,
				Resultado: models.ResultadoExito, OcurridoEn: time.Now(), Detalle: &detalle},
			{ID: uuid.New(), Accion: "auth.login_fallido", TipoRecurso: "personal", RecursoID: &canal, Resultado: models.ResultadoDenegado},
		},
		personal: map[uuid.UUID]*models.Personal{ana: {ID: ana, NombreLaboral: "Ana Castro", RolID: rol}},
		roles:    map[uuid.UUID]*models.Rol{rol: {ID: rol, Codigo: "Psicología"}},
	}
	svc := New(&fakeUOW{r}, "secret")
	actor := repository.ActorContext{ActorID: uuid.New()}

	page, err := svc.Logs(context.Background(), actor, Query{RecordID: &exp}, repository.PageRequest{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if r.filtro.RecursoID == nil || *r.filtro.RecursoID != exp || r.filtro.TipoRecurso != "expediente" {
		t.Fatalf("filtro: %+v", r.filtro)
	}
	a, b := page.Items[0], page.Items[1]
	if a.Actor == nil || a.Actor.NombreLaboral != "Ana Castro" || a.Rol == nil || *a.Rol != "Psicología" ||
		a.ExpedienteID == nil || *a.ExpedienteID != exp || a.Detalle == nil || *a.Detalle != detalle {
		t.Fatalf("entrada con actor: %+v", a)
	}
	if b.Actor != nil || b.Rol != nil || b.ExpedienteID != nil {
		t.Fatalf("entrada anónima de otro recurso: %+v", b)
	}
	if last := r.eventos[len(r.eventos)-1]; last.Accion != "audit.logs.list" || *last.ActorID != actor.ActorID {
		t.Fatalf("la consulta debía auditarse: %+v", last)
	}
}
