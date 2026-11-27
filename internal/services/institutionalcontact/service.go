// Package institutionalcontact implementa el directorio publico de ayuda.
package institutionalcontact

import (
	"context"
	"sort"

	"backend-hackathon/internal/models"
	icmodels "backend-hackathon/internal/models/institutionalcontact"
	repository "backend-hackathon/internal/repository/repositories"
)

type Service interface {
	List(ctx context.Context) ([]icmodels.Contact, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

func Order(insts []models.Institucion) []icmodels.Contact {
	out := make([]icmodels.Contact, 0, len(insts))
	for _, x := range insts {
		out = append(out, icmodels.Contact{
			ID: x.ID, Nombre: x.Nombre, Descripcion: x.Descripcion,
			Telefono: x.Telefono, Horario: x.Horario, Prioritario: x.Prioritario,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Prioritario && !out[j].Prioritario })
	return out
}

func (s *service) List(ctx context.Context) ([]icmodels.Contact, error) {
	insts, err := s.uow.Repositories().Instituciones().ListPublicables(ctx)
	if err != nil {
		return nil, err
	}
	return Order(insts), nil
}
