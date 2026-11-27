package note

import (
	"context"
	"errors"
	"testing"
	"time"

	casemodels "backend-hackathon/internal/models/casefile"
	nmodels "backend-hackathon/internal/models/note"
	repository "backend-hackathon/internal/repository/repositories"
	casesvc "backend-hackathon/internal/services/casefile"

	"github.com/google/uuid"
)

// fakeCases responde CreateNote según el texto de la nota.
type fakeCases struct {
	casesvc.Service
	calls int
}

func (f *fakeCases) CreateNote(_ context.Context, _ repository.ActorContext, _, _, _ uuid.UUID, in casemodels.NoteCreateRequest) (*casemodels.NoteView, error) {
	f.calls++
	switch in.Texto {
	case "cerrado":
		return nil, casesvc.ErrCaseClosed
	case "sin acceso":
		return nil, casesvc.ErrNotFound
	case "choque":
		return nil, repository.ErrConflict
	case "sin cifrado":
		return nil, casesvc.ErrCryptoUnavailable
	case "falla":
		return nil, errors.New("db caída")
	}
	return &casemodels.NoteView{ID: uuid.New(), Texto: in.Texto}, nil
}

func note(texto string) nmodels.SyncNote {
	return nmodels.SyncNote{OperationID: uuid.NewString(), RecordID: uuid.NewString(), DepartmentID: uuid.NewString(),
		Texto: texto, OcurridoEn: time.Now()}
}

func TestSyncResultsPerNote(t *testing.T) {
	cases := &fakeCases{}
	svc := New(cases)
	malRecord := note("ok")
	malRecord.RecordID = "no-uuid"
	in := nmodels.SyncRequest{Notes: []nmodels.SyncNote{
		note("ok"), note("cerrado"), note("sin acceso"), note("choque"), note("sin cifrado"), note("falla"), malRecord,
	}}
	res, err := svc.Sync(context.Background(), repository.ActorContext{ActorID: uuid.New()}, in)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	want := []struct{ result, code string }{
		{nmodels.ResultApplied, ""},
		{nmodels.ResultBlocked, "CASE_CLOSED"},
		{nmodels.ResultBlocked, "RESOURCE_NOT_FOUND"},
		{nmodels.ResultConflict, "CONFLICT"},
		{nmodels.ResultRetryable, "SERVICE_UNAVAILABLE"},
		{nmodels.ResultRetryable, "INTERNAL_ERROR"},
		{nmodels.ResultBlocked, "VALIDATION_ERROR"},
	}
	if len(res.Results) != len(want) {
		t.Fatalf("resultados = %d", len(res.Results))
	}
	for i, w := range want {
		got := res.Results[i]
		if got.Result != w.result || got.Code != w.code || got.OperationID != in.Notes[i].OperationID {
			t.Errorf("nota %d: %+v, quiero %+v", i, got, w)
		}
	}
	if res.Results[0].NoteID == nil {
		t.Error("applied debía traer noteId")
	}
	if cases.calls != 6 {
		t.Errorf("CreateNote llamado %d veces; la nota con recordId inválido no debía llegar", cases.calls)
	}
}

func TestSyncRejectsBadBatch(t *testing.T) {
	svc := New(&fakeCases{})
	dup := note("a")
	twin := note("b")
	twin.OperationID = dup.OperationID
	bad := note("c")
	bad.OperationID = "x"
	big := make([]nmodels.SyncNote, nmodels.MaxBatch+1)
	for i := range big {
		big[i] = note("n")
	}
	for name, notes := range map[string][]nmodels.SyncNote{
		"vacío":                nil,
		"más de 20":            big,
		"operationId repetido": {dup, twin},
		"operationId inválido": {bad},
	} {
		t.Run(name, func(t *testing.T) {
			var ve *ValidationError
			if _, err := svc.Sync(context.Background(), repository.ActorContext{}, nmodels.SyncRequest{Notes: notes}); !errors.As(err, &ve) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
