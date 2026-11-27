package course

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	cmodels "backend-hackathon/internal/models/course"
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
	cursos  []models.Curso
	audited []string
}

type fakeCursos struct {
	repository.CursoRepository
	r *fakeRepos
}

func (f fakeCursos) Create(_ context.Context, c *models.Curso) error {
	f.r.cursos = append(f.r.cursos, *c)
	return nil
}

func (f fakeCursos) GetByID(_ context.Context, id uuid.UUID) (*models.Curso, error) {
	for i := range f.r.cursos {
		if f.r.cursos[i].ID == id {
			return &f.r.cursos[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f fakeCursos) Inscritas(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{ids[0]: 3}, nil
}

type fakeAudit struct {
	repository.EventoAuditoriaRepository
	r *fakeRepos
}

func (f fakeAudit) Append(_ context.Context, e *models.EventoAuditoria) error {
	f.r.audited = append(f.r.audited, e.Accion)
	return nil
}

func (r *fakeRepos) Cursos() repository.CursoRepository                     { return fakeCursos{r: r} }
func (r *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository { return fakeAudit{r: r} }

func TestCreateYGet(t *testing.T) {
	r := &fakeRepos{}
	svc := &service{uow: &fakeUOW{r}, secret: "s", now: func() time.Time { return time.Date(2026, 11, 26, 18, 0, 0, 0, time.UTC) }}
	actor := repository.ActorContext{ActorID: uuid.New()}
	in := cmodels.CreateRequest{Tipo: models.CursoCurso, Titulo: "Finanzas para tu negocio", Descripcion: "Taller práctico",
		Fecha: "2026-12-12", Hora: "17:00", Lugar: "Sala 2, Los Mochis", Cupo: 25}

	c, err := svc.Create(context.Background(), actor, in)
	if err != nil || c.Fecha != "2026-12-12" || c.Inscritas != 0 || len(r.audited) != 1 {
		t.Fatalf("Create: %+v %v %v", c, err, r.audited)
	}
	got, err := svc.Get(context.Background(), c.ID)
	if err != nil || got.Inscritas != 3 || got.Titulo != in.Titulo {
		t.Fatalf("Get: %+v %v", got, err)
	}
	if _, err := svc.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inexistente: %v", err)
	}
}

func TestCreateValidacion(t *testing.T) {
	svc := &service{uow: &fakeUOW{&fakeRepos{}}, secret: "s", now: func() time.Time { return time.Date(2026, 11, 26, 18, 0, 0, 0, time.UTC) }}
	var he *httpx.Error
	_, err := svc.Create(context.Background(), repository.ActorContext{}, cmodels.CreateRequest{Tipo: "taller",
		Fecha: "2026-11-25", Hora: "25:00", Cupo: 0})
	if !errors.As(err, &he) {
		t.Fatalf("err = %v", err)
	}
	for _, f := range []string{"tipo", "titulo", "descripcion", "lugar", "fecha", "hora", "cupo"} {
		if he.Fields[f] == "" {
			t.Errorf("falta fields.%s: %v", f, he.Fields)
		}
	}
	if he.Fields["fecha"] != "fecha_pasada" {
		t.Errorf("fecha pasada: %v", he.Fields["fecha"])
	}
}
