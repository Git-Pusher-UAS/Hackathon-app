// Package helpevent implementa POST /api/help-events.
package helpevent

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	hemodels "backend-hackathon/internal/models/helpevent"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	maxSkew = 5 * time.Minute
	maxAge  = 30 * 24 * time.Hour
)

type Service interface {
	Record(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in hemodels.CreateRequest) error
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

func (s *service) validate(opID uuid.UUID, in hemodels.CreateRequest) (*uuid.UUID, error) {
	fields := map[string]string{}
	if in.Action != models.AccionAyudaDialerOpened {
		fields["action"] = "valor_invalido"
	}
	now := s.now()
	if in.OccurredAt.IsZero() || in.OccurredAt.After(now.Add(maxSkew)) || in.OccurredAt.Before(now.Add(-maxAge)) {
		fields["occurredAt"] = "fuera_de_rango"
	}
	if in.OperationID != "" && !strings.EqualFold(in.OperationID, opID.String()) {
		fields["operationId"] = "distinto_de_clave_idempotente"
	}
	var contact *uuid.UUID
	if in.ContactID != nil && strings.TrimSpace(*in.ContactID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*in.ContactID))
		if err != nil {
			fields["contactId"] = "formato_invalido"
		} else {
			contact = &id
		}
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("evento de ayuda invalido", fields)
	}
	return contact, nil
}

func (s *service) Record(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in hemodels.CreateRequest) error {
	contact, err := s.validate(opID, in)
	if err != nil {
		return err
	}
	return s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		prev, err := r.EventosAyuda().GetByOperacionID(ctx, opID)
		if err == nil {
			if prev.PersonalID != actor.ActorID {
				return httpx.E(fiber.StatusConflict, "IDEMPOTENCY_MISMATCH", "la clave ya se uso")
			}
			return nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		return r.EventosAyuda().Create(ctx, &models.EventoLocalAyuda{
			PersonalID: actor.ActorID, ContactoID: contact,
			Accion: in.Action, OcurridoEn: in.OccurredAt.UTC(), OperacionID: opID,
		})
	})
}
