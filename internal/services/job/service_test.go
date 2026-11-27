package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
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
	vac  *stubVacRepo
	post *stubPostRepo
	cv   *stubCVRepo
}

func (r *stubRepos) Vacantes() repository.VacanteRepository          { return r.vac }
func (r *stubRepos) Postulaciones() repository.PostulacionRepository { return r.post }
func (r *stubRepos) CV() repository.CVRepository                     { return r.cv }

type stubVacRepo struct {
	repository.VacanteRepository
	byID map[uuid.UUID]*models.Vacante
}

func (r *stubVacRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Vacante, error) {
	v, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *v
	return &cp, nil
}
func (r *stubVacRepo) List(_ context.Context, _ repository.VacanteFilter, _ repository.PageRequest) (repository.Page[models.Vacante], error) {
	out := []models.Vacante{}
	for _, v := range r.byID {
		out = append(out, *v)
	}
	return repository.Page[models.Vacante]{Items: out}, nil
}
func (r *stubVacRepo) CountPostulaciones(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}

type stubPostRepo struct {
	repository.PostulacionRepository
	exists  bool
	created *models.Postulacion
}

func (r *stubPostRepo) Exists(_ context.Context, _, _ uuid.UUID) (bool, error) { return r.exists, nil }
func (r *stubPostRepo) ExistsMany(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (map[uuid.UUID]bool, error) {
	return map[uuid.UUID]bool{}, nil
}
func (r *stubPostRepo) Create(_ context.Context, p *models.Postulacion) error {
	cp := *p
	r.created = &cp
	return nil
}

type stubCVRepo struct {
	repository.CVRepository
	perfil *models.CVPerfil
	exps   []models.CVExperiencia
	eds    []models.CVEstudio
}

func (r *stubCVRepo) GetPerfil(_ context.Context, _ uuid.UUID) (*models.CVPerfil, error) {
	if r.perfil == nil {
		return nil, repository.ErrNotFound
	}
	cp := *r.perfil
	return &cp, nil
}
func (r *stubCVRepo) ListExperiencias(_ context.Context, _ uuid.UUID) ([]models.CVExperiencia, error) {
	return r.exps, nil
}
func (r *stubCVRepo) ListEstudios(_ context.Context, _ uuid.UUID) ([]models.CVEstudio, error) {
	return r.eds, nil
}

func TestApplyRejectsIncompleteCV(t *testing.T) {
	vacID, usuaria := uuid.New(), uuid.New()
	repos := &stubRepos{
		vac:  &stubVacRepo{byID: map[uuid.UUID]*models.Vacante{vacID: {ID: vacID, Estado: models.VacantePublicada}}},
		post: &stubPostRepo{},
		cv:   &stubCVRepo{perfil: &models.CVPerfil{Nombre: "Ana", Habilidades: "[]"}},
	}
	svc := New(&stubUOW{repos: repos})
	svc.(*service).now = func() time.Time { return time.Now() }
	err := svc.Apply(context.Background(), vacID, usuaria)
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 422 {
		t.Fatalf("expected 422 validation error, got %v", err)
	}
	if repos.post.created != nil {
		t.Fatalf("should not create application")
	}
}

func TestApplyRejectsUnpublishedVacancy(t *testing.T) {
	vacID, usuaria := uuid.New(), uuid.New()
	repos := &stubRepos{
		vac:  &stubVacRepo{byID: map[uuid.UUID]*models.Vacante{vacID: {ID: vacID, Estado: models.VacantePendiente}}},
		post: &stubPostRepo{},
		cv:   &stubCVRepo{perfil: &models.CVPerfil{Nombre: "Ana", Habilidades: "[\"Go\"]"}},
	}
	svc := New(&stubUOW{repos: repos})
	svc.(*service).now = func() time.Time { return time.Now() }
	err := svc.Apply(context.Background(), vacID, usuaria)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetHidesUnpublishedVacancy(t *testing.T) {
	vacID := uuid.New()
	repos := &stubRepos{
		vac:  &stubVacRepo{byID: map[uuid.UUID]*models.Vacante{vacID: {ID: vacID, Estado: models.VacantePendiente}}},
		post: &stubPostRepo{},
		cv:   &stubCVRepo{},
	}
	svc := New(&stubUOW{repos: repos})
	_, err := svc.Get(context.Background(), vacID, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
