// Package helpevent implementa POST /help-events (API-040).
package helpevent

import (
	"context"
	"errors"
	"time"

	"backend-hackathon/internal/models"
	hemodels "backend-hackathon/internal/models/helpevent"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

// ErrInvalid indica un cuerpo que no cumple el contrato.
type ErrInvalid struct{ Msg string }

func (e *ErrInvalid) Error() string { return e.Msg }

// maxSkew tolera relojes de dispositivo adelantados.
const maxSkew = 5 * time.Minute

// maxAge limita eventos guardados offline demasiado antiguos.
const maxAge = 30 * 24 * time.Hour

type Service interface {
	Record(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in hemodels.CreateRequest) error
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

func (s *service) Record(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in hemodels.CreateRequest) error {
	if in.Action != models.AccionAyudaDialerOpened {
		return &ErrInvalid{Msg: "action debe ser dialer_opened"}
	}
	now := s.now()
	if in.OccurredAt.IsZero() || in.OccurredAt.After(now.Add(maxSkew)) || in.OccurredAt.Before(now.Add(-maxAge)) {
		return &ErrInvalid{Msg: "occurredAt fuera de rango"}
	}
	return s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		prev, err := r.EventosAyuda().GetByOperacionID(ctx, opID)
		if err == nil {
			// Reintento: misma operación del mismo actor no hace nada.
			if prev.PersonalID != actor.ActorID {
				return repository.ErrConflict
			}
			return nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		return r.EventosAyuda().Create(ctx, &models.EventoLocalAyuda{
			PersonalID: actor.ActorID, Accion: in.Action, OcurridoEn: in.OccurredAt.UTC(), OperacionID: opID})
	})
}
