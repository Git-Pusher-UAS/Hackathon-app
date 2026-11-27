package note

import (
	"context"
	"errors"
	"strconv"

	"backend-hackathon/internal/httpx"
	casemodels "backend-hackathon/internal/models/casefile"
	nmodels "backend-hackathon/internal/models/note"
	repository "backend-hackathon/internal/repository/repositories"
	casesvc "backend-hackathon/internal/services/casefile"

	"github.com/google/uuid"
)

type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

type Service interface {
	Sync(ctx context.Context, actor repository.ActorContext, in nmodels.SyncRequest) ([]nmodels.SyncResult, error)
}

type service struct{ cases casesvc.Service }

func New(cases casesvc.Service) Service { return &service{cases: cases} }

func (s *service) Sync(ctx context.Context, actor repository.ActorContext, in nmodels.SyncRequest) ([]nmodels.SyncResult, error) {
	if len(in.Notes) < 1 || len(in.Notes) > nmodels.MaxBatch {
		return nil, &ValidationError{Msg: "notes debe tener entre 1 y 20 elementos"}
	}
	ops := make([]uuid.UUID, len(in.Notes))
	seen := map[uuid.UUID]bool{}
	for i, n := range in.Notes {
		op, err := uuid.Parse(n.OperationID)
		if err != nil {
			return nil, &ValidationError{Msg: "operationId invalido en la posicion " + strconv.Itoa(i)}
		}
		if seen[op] {
			return nil, &ValidationError{Msg: "operationId repetido en el lote"}
		}
		seen[op] = true
		ops[i] = op
	}

	out := make([]nmodels.SyncResult, 0, len(in.Notes))
	for i, n := range in.Notes {
		res := nmodels.SyncResult{ID: ops[i].String()}
		recordID, err1 := uuid.Parse(n.RecordID)
		depID, err2 := uuid.Parse(n.DepartmentID)
		if err1 != nil || err2 != nil {
			res.Result, res.Reason = nmodels.ResultBlocked, "validation_failed"
			out = append(out, res)
			continue
		}
		view, err := s.cases.CreateNote(ctx, actor, recordID, depID, ops[i], casemodels.NoteCreateRequest{Texto: n.Texto, OcurridoEn: n.OcurridoEn, CorrigeNotaID: n.CorrigeNotaID})
		if err == nil {
			res.Result, res.NoteID = nmodels.ResultApplied, &view.ID
		} else {
			res.Result, res.Reason = classify(err)
		}
		out = append(out, res)
	}
	return out, nil
}

func classify(err error) (string, string) {
	var ve *casesvc.ValidationError
	var he *httpx.Error
	switch {
	case errors.As(err, &ve), errors.As(err, &he):
		return nmodels.ResultBlocked, "validation_failed"
	case errors.Is(err, casesvc.ErrNotFound):
		return nmodels.ResultBlocked, "assignment_revoked"
	case errors.Is(err, casesvc.ErrCaseClosed):
		return nmodels.ResultConflict, "record_closed"
	case errors.Is(err, casesvc.ErrAlreadyCorrected):
		return nmodels.ResultConflict, "already_corrected"
	case errors.Is(err, repository.ErrConflict):
		return nmodels.ResultConflict, "conflict"
	case errors.Is(err, casesvc.ErrCryptoUnavailable):
		return nmodels.ResultRetryable, "encryption_unavailable"
	default:
		return nmodels.ResultRetryable, "internal_error"
	}
}
