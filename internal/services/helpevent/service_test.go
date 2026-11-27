package helpevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	hemodels "backend-hackathon/internal/models/helpevent"
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
	ev *fakeEventos
}

func (r *fakeRepos) EventosAyuda() repository.EventoLocalAyudaRepository { return r.ev }

type fakeEventos struct {
	repository.EventoLocalAyudaRepository
	byOp map[uuid.UUID]models.EventoLocalAyuda
}

func (f *fakeEventos) GetByOperacionID(_ context.Context, op uuid.UUID) (*models.EventoLocalAyuda, error) {
	if e, ok := f.byOp[op]; ok {
		return &e, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeEventos) Create(_ context.Context, e *models.EventoLocalAyuda) error {
	f.byOp[e.OperacionID] = *e
	return nil
}

func TestRecord(t *testing.T) {
	ev := &fakeEventos{byOp: map[uuid.UUID]models.EventoLocalAyuda{}}
	svc := New(&fakeUOW{&fakeRepos{ev: ev}})
	actor := repository.ActorContext{ActorID: uuid.New()}
	op := uuid.New()
	ok := hemodels.CreateRequest{Action: models.AccionAyudaDialerOpened, OccurredAt: time.Now().Add(-time.Hour)}

	if err := svc.Record(context.Background(), actor, op, ok); err != nil {
		t.Fatalf("Record: %v", err)
	}
	// Reintento con la misma operación: no duplica ni falla.
	if err := svc.Record(context.Background(), actor, op, ok); err != nil {
		t.Fatalf("reintento: %v", err)
	}
	if len(ev.byOp) != 1 || ev.byOp[op].PersonalID != actor.ActorID {
		t.Fatalf("eventos: %+v", ev.byOp)
	}
	// La misma operación desde otra cuenta es un conflicto.
	if err := svc.Record(context.Background(), repository.ActorContext{ActorID: uuid.New()}, op, ok); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("otra cuenta: err = %v", err)
	}

	var inv *ErrInvalid
	for name, req := range map[string]hemodels.CreateRequest{
		"acción desconocida": {Action: "sms_sent", OccurredAt: time.Now()},
		"sin fecha":          {Action: models.AccionAyudaDialerOpened},
		"en el futuro":       {Action: models.AccionAyudaDialerOpened, OccurredAt: time.Now().Add(time.Hour)},
		"demasiado antigua":  {Action: models.AccionAyudaDialerOpened, OccurredAt: time.Now().Add(-60 * 24 * time.Hour)},
	} {
		if err := svc.Record(context.Background(), actor, uuid.New(), req); !errors.As(err, &inv) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
