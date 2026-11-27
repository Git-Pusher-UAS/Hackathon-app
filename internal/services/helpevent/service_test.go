package helpevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
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
	contact := uuid.NewString()
	ok := hemodels.CreateRequest{ContactID: &contact, Accion: models.AccionAyudaDialerOpened,
		OcurridoEn: time.Now().Add(-time.Hour), OperationID: op.String()}

	if err := svc.Record(context.Background(), actor, op, ok); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := svc.Record(context.Background(), actor, op, ok); err != nil {
		t.Fatalf("reintento: %v", err)
	}
	got := ev.byOp[op]
	if len(ev.byOp) != 1 || got.PersonalID != actor.ActorID || got.ContactoID == nil || got.ContactoID.String() != contact {
		t.Fatalf("eventos: %+v", ev.byOp)
	}
	var he *httpx.Error
	if err := svc.Record(context.Background(), repository.ActorContext{ActorID: uuid.New()}, op, ok); !errors.As(err, &he) || he.Code != "idempotency_mismatch" {
		t.Fatalf("otra cuenta: %v", err)
	}
}

func TestRecordValidation(t *testing.T) {
	svc := New(&fakeUOW{&fakeRepos{ev: &fakeEventos{byOp: map[uuid.UUID]models.EventoLocalAyuda{}}}})
	bad := "x"
	cases := map[string]struct {
		req   hemodels.CreateRequest
		field string
	}{
		"acción desconocida": {hemodels.CreateRequest{Accion: "sms_sent", OcurridoEn: time.Now()}, "accion"},
		"sin fecha":          {hemodels.CreateRequest{Accion: models.AccionAyudaDialerOpened}, "ocurridoEn"},
		"en el futuro":       {hemodels.CreateRequest{Accion: models.AccionAyudaDialerOpened, OcurridoEn: time.Now().Add(time.Hour)}, "ocurridoEn"},
		"operationId ajeno":  {hemodels.CreateRequest{Accion: models.AccionAyudaDialerOpened, OcurridoEn: time.Now(), OperationID: uuid.NewString()}, "operationId"},
		"contactId inválido": {hemodels.CreateRequest{Accion: models.AccionAyudaDialerOpened, OcurridoEn: time.Now(), ContactID: &bad}, "contactId"},
	}
	for name, tc := range cases {
		var he *httpx.Error
		err := svc.Record(context.Background(), repository.ActorContext{ActorID: uuid.New()}, uuid.New(), tc.req)
		if !errors.As(err, &he) || he.Status != 422 || he.Fields[tc.field] == "" {
			t.Errorf("%s: %v", name, err)
		}
	}
}
