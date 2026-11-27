package auditalert

import (
	"context"
	"errors"

	"backend-hackathon/internal/models"
	aamodels "backend-hackathon/internal/models/auditalert"
	repository "backend-hackathon/internal/repository/repositories"
)

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoAlerta, page repository.PageRequest) ([]aamodels.Item, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

func (s *service) List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoAlerta, page repository.PageRequest) ([]aamodels.Item, error) {
	out := []aamodels.Item{}
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.AlertasAuditoria().List(ctx, repository.AlertaFilter{Estado: estado}, page)
		if err != nil {
			return err
		}
		out = make([]aamodels.Item, 0, len(rp.Items))
		for _, a := range rp.Items {
			item := aamodels.Item{ID: a.ID, Estado: a.Estado, Fecha: a.CreadoEn.UTC()}
			if a.EventoDisparadorID != nil {
				ev, err := r.EventosAuditoria().GetByID(ctx, *a.EventoDisparadorID)
				if err == nil && ev.TipoRecurso == "expediente" {
					item.ExpedienteID = ev.RecursoID
				} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
					return err
				}
			}
			out = append(out, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
