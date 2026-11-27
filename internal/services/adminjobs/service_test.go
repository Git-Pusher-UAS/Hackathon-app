package adminjobs

import (
	"context"
	"encoding/json"
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

func (u *stubUOW) Do(_ context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(context.Background(), u.repos)
}
func (u *stubUOW) Repositories() repository.Repositories { return u.repos }

type stubRepos struct {
	repository.Repositories
	post *stubPostRepo
	aud  *stubAuditRepo
}

func (r *stubRepos) Postulaciones() repository.PostulacionRepository        { return r.post }
func (r *stubRepos) EventosAuditoria() repository.EventoAuditoriaRepository { return r.aud }

type stubPostRepo struct {
	repository.PostulacionRepository
	item *models.Postulacion
}

func (r *stubPostRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Postulacion, error) {
	if r.item == nil || r.item.ID != id {
		return nil, repository.ErrNotFound
	}
	cp := *r.item
	return &cp, nil
}
func (r *stubPostRepo) Save(_ context.Context, p *models.Postulacion) error {
	cp := *p
	r.item = &cp
	return nil
}

type stubAuditRepo struct {
	repository.EventoAuditoriaRepository
	events []*models.EventoAuditoria
}

func (r *stubAuditRepo) Append(_ context.Context, e *models.EventoAuditoria) error {
	cp := *e
	r.events = append(r.events, &cp)
	return nil
}

func TestContactReleaseRequiresMotivo(t *testing.T) {
	svc := New(&stubUOW{repos: &stubRepos{post: &stubPostRepo{}, aud: &stubAuditRepo{}}})
	_, err := svc.ContactRelease(context.Background(), repository.ActorContext{ActorID: uuid.New()}, uuid.New(), "  ", nil)
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 422 {
		t.Fatalf("expected 422, got %v", err)
	}
}

func TestContactReleaseAuditsMetadata(t *testing.T) {
	appID, actorID, jobID, candID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	post := &models.Postulacion{ID: appID, VacanteID: jobID, UsuariaID: candID, Estado: models.PostulacionContratada}
	aud := &stubAuditRepo{}
	svc := New(&stubUOW{repos: &stubRepos{post: &stubPostRepo{item: post}, aud: aud}})
	svc.now = func() time.Time { return time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC) }
	_, err := svc.ContactRelease(context.Background(), repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}, appID, "verificacion", []string{"nombre"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(aud.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(aud.events))
	}
	ev := aud.events[0]
	if ev.Accion != "admin.contact_release" || ev.TipoRecurso != "postulacion" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	var meta map[string]any
	if err := json.Unmarshal(ev.Metadatos, &meta); err != nil {
		t.Fatalf("meta: %v", err)
	}
	if meta["motivo"] != "verificacion" {
		t.Fatalf("meta motivo: %v", meta["motivo"])
	}
	if meta["applicationId"] != appID.String() {
		t.Fatalf("meta app: %v", meta["applicationId"])
	}
}
