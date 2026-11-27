// Package helpevent implementa POST /help-events (§9.2).
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

	"github.com/google/uuid"
)

const (
	// maxSkew tolera relojes de dispositivo adelantados.
	maxSkew = 5 * time.Minute
	// maxAge limita eventos guardados sin conexión demasiado antiguos.
	maxAge = 30 * 24 * time.Hour
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
	if in.Accion != models.AccionAyudaDialerOpened {
		fields["accion"] = "valor_invalido"
	}
	now := s.now()
	if in.OcurridoEn.IsZero() || in.OcurridoEn.After(now.Add(maxSkew)) || in.OcurridoEn.Before(now.Add(-maxAge)) {
		fields["ocurridoEn"] = "fuera_de_rango"
	}
	if in.OperationID != "" && !strings.EqualFold(in.OperationID, opID.String()) {
		fields["operationId"] = "distinto_de_idempotency_key"
	}
	var contact *uuid.UUID
	if in.ContactID != nil {
		id, err := uuid.Parse(*in.ContactID)
		if err != nil {
			fields["contactId"] = "formato_invalido"
		} else {
			contact = &id
		}
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("evento de ayuda inválido", fields)
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
			// Reintento de la misma cuenta: no hace nada.
			if prev.PersonalID != actor.ActorID {
				return httpx.E(409, "idempotency_mismatch", "la clave ya se usó")
			}
			return nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		return r.EventosAyuda().Create(ctx, &models.EventoLocalAyuda{PersonalID: actor.ActorID, ContactoID: contact,
			Accion: in.Accion, OcurridoEn: in.OcurridoEn.UTC(), OperacionID: opID})
	})
}
