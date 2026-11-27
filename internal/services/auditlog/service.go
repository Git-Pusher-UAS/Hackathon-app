package auditlog

import (
	"context"
	"errors"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/models"
	almodels "backend-hackathon/internal/models/auditlog"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

type Query struct {
	ActorID  *uuid.UUID
	RecordID *uuid.UUID
	Action   string
	Rango    repository.TimeRange
}

type Service interface {
	Logs(ctx context.Context, actor repository.ActorContext, q Query, page repository.PageRequest) ([]almodels.Entry, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

type actorInfo struct {
	actor *almodels.Actor
	rol   *string
}

func (s *service) Logs(ctx context.Context, actor repository.ActorContext, q Query, page repository.PageRequest) ([]almodels.Entry, error) {
	f := repository.AuditoriaFilter{ActorID: q.ActorID, Accion: q.Action, Rango: q.Rango}
	if q.RecordID != nil {
		f.RecursoID, f.TipoRecurso = q.RecordID, "expediente"
	}
	out := []almodels.Entry{}
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.EventosAuditoria().List(ctx, f, page)
		if err != nil {
			return err
		}
		actores := map[uuid.UUID]actorInfo{}
		roles := map[uuid.UUID]string{}
		out = make([]almodels.Entry, 0, len(rp.Items))
		for _, e := range rp.Items {
			item := almodels.Entry{ID: e.ID, Fecha: e.OcurridoEn.UTC(), Accion: e.Accion}
			if e.TipoRecurso == "expediente" {
				item.Expediente = e.RecursoID
			}
			if e.ActorID != nil {
				info, ok := actores[*e.ActorID]
				if !ok {
					p, err := r.Personal().GetByID(ctx, *e.ActorID)
					switch {
					case err == nil:
						info.actor = &almodels.Actor{ID: p.ID, NombreLaboral: p.NombreLaboral}
						rol, ok := roles[p.RolID]
						if !ok {
							if rr, err := r.Roles().GetByID(ctx, p.RolID); err == nil {
								rol = rr.Codigo
							}
							roles[p.RolID] = rol
						}
						if rol != "" {
							info.rol = &rol
						}
					case !errors.Is(err, repository.ErrNotFound):
						return err
					}
					actores[*e.ActorID] = info
				}
				item.Actor, item.Rol = info.actor, info.rol
			}
			out = append(out, item)
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "audit.logs.list", "auditoria", models.ResultadoExito, nil, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
