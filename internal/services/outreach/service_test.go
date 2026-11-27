package outreach

import (
	"context"
	"errors"
	"testing"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	omodels "backend-hackathon/internal/models/outreach"
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
	sector  uuid.UUID
	visitas []models.Visita
}

type fakeSectores struct {
	repository.SectorRepository
	r *fakeRepos
}

func (f fakeSectores) GetByID(_ context.Context, id uuid.UUID) (*models.Sector, error) {
	if id == f.r.sector {
		return &models.Sector{ID: id, Nombre: "Sector 12"}, nil
	}
	return nil, repository.ErrNotFound
}

type fakeVisitas struct {
	repository.VisitaRepository
	r *fakeRepos
}

func (f fakeVisitas) Create(_ context.Context, v *models.Visita) error {
	f.r.visitas = append(f.r.visitas, *v)
	return nil
}

func (f fakeVisitas) List(context.Context, *models.EstadoVisita, repository.PageRequest) (repository.Page[models.Visita], error) {
	return repository.Page[models.Visita]{Items: f.r.visitas}, nil
}

type fakePersonal struct{ repository.PersonalRepository }

func (fakePersonal) GetByID(context.Context, uuid.UUID) (*models.Personal, error) {
	return nil, repository.ErrNotFound
}

type fakeAudit struct {
	repository.EventoAuditoriaRepository
}

func (fakeAudit) Append(context.Context, *models.EventoAuditoria) error { return nil }

func (r *fakeRepos) Sectores() repository.SectorRepository                  { return fakeSectores{r: r} }
func (r *fakeRepos) Visitas() repository.VisitaRepository                   { return fakeVisitas{r: r} }
func (r *fakeRepos) Personal() repository.PersonalRepository                { return fakePersonal{} }
func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository { return fakeAudit{} }

func TestVisitas(t *testing.T) {
	r := &fakeRepos{sector: uuid.New()}
	svc := New(&fakeUOW{r}, "s")
	actor := repository.ActorContext{ActorID: uuid.New()}

	v, err := svc.Create(context.Background(), actor, omodels.CreateRequest{SectorID: r.sector.String(), Fecha: "2026-12-03", Tema: "Prevención de la violencia"})
	if err != nil || v.Estado != models.VisitaPlaneada || v.Sector != "Sector 12" || v.Responsable != "Capacitación" || v.Fecha != "2026-12-03" {
		t.Fatalf("Create: %+v %v", v, err)
	}
	page, err := svc.List(context.Background(), actor, nil, repository.PageRequest{Limit: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].Sector != "Sector 12" {
		t.Fatalf("List: %+v %v", page, err)
	}
	var he *httpx.Error
	if _, err := svc.Create(context.Background(), actor, omodels.CreateRequest{SectorID: uuid.NewString(), Fecha: "2026-12-03", Tema: "x"}); !errors.As(err, &he) || he.Fields["sectorId"] != "no_existe" {
		t.Fatalf("sector inexistente: %v", err)
	}
	if _, err := svc.Create(context.Background(), actor, omodels.CreateRequest{SectorID: "x", Fecha: "3 dic"}); !errors.As(err, &he) || len(he.Fields) != 3 {
		t.Fatalf("validación: %v", err)
	}
}
