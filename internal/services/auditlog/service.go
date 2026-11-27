// Package auditlog implementa la consulta de la bitácora (§13.1). La consulta
// misma queda registrada en la bitácora.
package auditlog

import (
	"context"
	"errors"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	almodels "backend-hackathon/internal/models/auditlog"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const cursorScope = "audit.logs"

// Query son los filtros de la bitácora.
type Query struct {
	ActorID  *uuid.UUID
	RecordID *uuid.UUID // expediente
	Action   string
	Rango    repository.TimeRange
}

type Service interface {
	Logs(ctx context.Context, actor repository.ActorContext, q Query, page repository.PageRequest) (*almodels.Page, error)
}

type service struct {
	uow    repository.UnitOfWork
	secret string
}

func New(uow repository.UnitOfWork, secret string) Service { return &service{uow: uow, secret: secret} }

func optID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

type actorInfo struct {
	actor *almodels.Actor
	rol   *string
}

func (s *service) Logs(ctx context.Context, actor repository.ActorContext, q Query, page repository.PageRequest) (*almodels.Page, error) {
	cur := httpx.Cursor{Secret: s.secret, Scope: cursorScope}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, optID(q.ActorID), optID(q.RecordID), q.Action,
		q.Rango.Desde.String(), q.Rango.Hasta.String())
	if err != nil {
		return nil, err
	}
	f := repository.AuditoriaFilter{ActorID: q.ActorID, Accion: q.Action, Rango: q.Rango}
	if q.RecordID != nil {
		f.RecursoID, f.TipoRecurso = q.RecordID, "expediente"
	}
	out := &almodels.Page{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.EventosAuditoria().List(ctx, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		actores := map[uuid.UUID]actorInfo{}
		roles := map[uuid.UUID]string{}
		out.Items = make([]almodels.Entry, 0, len(rp.Items))
		for _, e := range rp.Items {
			item := almodels.Entry{ID: e.ID, Fecha: e.OcurridoEn.UTC(), Accion: e.Accion, Resultado: e.Resultado,
				Detalle: e.Detalle, RequestID: e.RequestID}
			if e.TipoRecurso == "expediente" {
				item.ExpedienteID = e.RecursoID
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
			out.Items = append(out.Items, item)
		}
		out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
		return audit.RecordResource(ctx, r, &actor.ActorID, "audit.logs.list", "auditoria",
			models.ResultadoExito, nil, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
