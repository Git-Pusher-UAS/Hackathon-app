// Package note implementa la sincronización de borradores offline (API-041).
// Cada nota se guarda con las mismas reglas que
// POST /case-files/{recordId}/departments/{departmentId}/notes, en su propia
// transacción, así que una nota rechazada no tumba el resto del lote.
package note

import (
	"context"
	"errors"
	"strconv"

	casemodels "backend-hackathon/internal/models/casefile"
	nmodels "backend-hackathon/internal/models/note"
	repository "backend-hackathon/internal/repository/repositories"
	casesvc "backend-hackathon/internal/services/casefile"

	"github.com/google/uuid"
)

// ValidationError invalida el lote completo.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

type Service interface {
	Sync(ctx context.Context, actor repository.ActorContext, in nmodels.SyncRequest) (*nmodels.SyncResponse, error)
}

type service struct{ cases casesvc.Service }

func New(cases casesvc.Service) Service { return &service{cases: cases} }

func (s *service) Sync(ctx context.Context, actor repository.ActorContext, in nmodels.SyncRequest) (*nmodels.SyncResponse, error) {
	if len(in.Notes) < 1 || len(in.Notes) > nmodels.MaxBatch {
		return nil, &ValidationError{Msg: "notes debe tener entre 1 y 20 elementos"}
	}
	ops := make([]uuid.UUID, len(in.Notes))
	seen := map[uuid.UUID]bool{}
	for i, n := range in.Notes {
		op, err := uuid.Parse(n.OperationID)
		if err != nil {
			return nil, &ValidationError{Msg: "operationId inválido en la posición " + strconv.Itoa(i)}
		}
		if seen[op] {
			return nil, &ValidationError{Msg: "operationId repetido en el lote"}
		}
		seen[op] = true
		ops[i] = op
	}

	out := &nmodels.SyncResponse{Results: make([]nmodels.SyncResult, 0, len(in.Notes))}
	for i, n := range in.Notes {
		res := nmodels.SyncResult{OperationID: ops[i].String()}
		recordID, err1 := uuid.Parse(n.RecordID)
		depID, err2 := uuid.Parse(n.DepartmentID)
		if err1 != nil || err2 != nil {
			res.Result, res.Code = nmodels.ResultBlocked, "VALIDATION_ERROR"
			out.Results = append(out.Results, res)
			continue
		}
		view, err := s.cases.CreateNote(ctx, actor, recordID, depID, ops[i], casemodels.NoteCreateRequest{
			Texto: n.Texto, OcurridoEn: n.OcurridoEn, CorrigeNotaID: n.CorrigeNotaID})
		if err == nil {
			res.Result, res.NoteID = nmodels.ResultApplied, &view.ID
		} else {
			res.Result, res.Code = classify(err)
		}
		out.Results = append(out.Results, res)
	}
	return out, nil
}

func classify(err error) (string, string) {
	var ve *casesvc.ValidationError
	switch {
	case errors.As(err, &ve):
		return nmodels.ResultBlocked, "VALIDATION_ERROR"
	case errors.Is(err, casesvc.ErrNotFound):
		return nmodels.ResultBlocked, "RESOURCE_NOT_FOUND"
	case errors.Is(err, casesvc.ErrCaseClosed):
		return nmodels.ResultBlocked, "CASE_CLOSED"
	case errors.Is(err, repository.ErrConflict):
		return nmodels.ResultConflict, "CONFLICT"
	case errors.Is(err, casesvc.ErrCryptoUnavailable):
		return nmodels.ResultRetryable, "SERVICE_UNAVAILABLE"
	default:
		return nmodels.ResultRetryable, "INTERNAL_ERROR"
	}
}
