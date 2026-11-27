// Package auditlog implementa la consulta de la bitácora (API-035) y de las
// alertas de acceso (API-036). Toda consulta queda a su vez auditada.
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

const (
	scopeLogs   = "audit.logs"
	scopeAlerts = "audit.alerts"
)

type Service interface {
	Logs(ctx context.Context, actor repository.ActorContext, f repository.AuditoriaFilter, page repository.PageRequest) (*almodels.EntryPage, error)
	Alerts(ctx context.Context, actor repository.ActorContext, f repository.AlertaFilter, page repository.PageRequest) (*almodels.AlertPage, error)
}

type service struct {
	uow    repository.UnitOfWork
	secret string
}

func New(uow repository.UnitOfWork, secret string) Service {
	return &service{uow: uow, secret: secret}
}

func optID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func optStr[T ~string](v *T) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func (s *service) Logs(ctx context.Context, actor repository.ActorContext, f repository.AuditoriaFilter, page repository.PageRequest) (*almodels.EntryPage, error) {
	cur := httpx.Cursor{Secret: s.secret, Scope: scopeLogs}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, optID(f.ActorID), f.Accion, f.TipoRecurso, optID(f.RecursoID),
		optStr(f.Resultado), f.Rango.Desde.String(), f.Rango.Hasta.String())
	if err != nil {
		return nil, err
	}
	out := &almodels.EntryPage{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.EventosAuditoria().List(ctx, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		usuarios := map[uuid.UUID]*string{}
		out.Items = make([]almodels.Entry, 0, len(rp.Items))
		for _, e := range rp.Items {
			item := almodels.Entry{ID: e.ID, Secuencia: e.Secuencia, ActorID: e.ActorID, Accion: e.Accion,
				TipoRecurso: e.TipoRecurso, RecursoID: e.RecursoID, Resultado: e.Resultado,
				OcurridoEn: e.OcurridoEn.UTC(), RequestID: e.RequestID}
			if e.ActorID != nil {
				u, ok := usuarios[*e.ActorID]
				if !ok {
					p, err := r.Personal().GetByID(ctx, *e.ActorID)
					switch {
					case err == nil:
						u = &p.Usuario
					case !errors.Is(err, repository.ErrNotFound):
						return err
					}
					usuarios[*e.ActorID] = u
				}
				item.ActorUsuario = u
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

func (s *service) Alerts(ctx context.Context, actor repository.ActorContext, f repository.AlertaFilter, page repository.PageRequest) (*almodels.AlertPage, error) {
	cur := httpx.Cursor{Secret: s.secret, Scope: scopeAlerts}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, optID(f.ActorObservadoID), optStr(f.Nivel), optStr(f.Estado),
		f.Rango.Desde.String(), f.Rango.Hasta.String())
	if err != nil {
		return nil, err
	}
	out := &almodels.AlertPage{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.AlertasAuditoria().List(ctx, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		out.Items = make([]almodels.Alert, 0, len(rp.Items))
		for _, a := range rp.Items {
			out.Items = append(out.Items, almodels.Alert{ID: a.ID, ActorObservadoID: a.ActorObservadoID, Tipo: a.Tipo,
				VentanaInicio: a.VentanaInicio.UTC(), VentanaFin: a.VentanaFin.UTC(), Conteo: a.Conteo, Umbral: a.Umbral,
				Nivel: a.Nivel, Estado: a.Estado, CreatedAt: a.CreadoEn.UTC()})
		}
		out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
		return audit.RecordResource(ctx, r, &actor.ActorID, "audit.alerts.list", "auditoria",
			models.ResultadoExito, nil, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
