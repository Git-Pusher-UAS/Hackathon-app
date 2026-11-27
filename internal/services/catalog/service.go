// Package catalog implementa GET /api/catalogs.
package catalog

import (
	"context"

	"backend-hackathon/internal/models"
	catmodels "backend-hackathon/internal/models/catalog"
	repository "backend-hackathon/internal/repository/repositories"
)

type Service interface {
	Catalogs(ctx context.Context) (*catmodels.Response, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

var origenes = []catmodels.Opcion{
	{Codigo: string(models.OrigenPresencial), Nombre: "Presencial"},
	{Codigo: string(models.OrigenInstitucion), Nombre: "Institucion"},
	{Codigo: string(models.OrigenLineaAtencion), Nombre: "Linea de atencion"},
	{Codigo: string(models.OrigenPatrullaGenero), Nombre: "Patrulla de genero"},
	{Codigo: string(models.OrigenEmergenciaReferida), Nombre: "Emergencia referida"},
}

func (s *service) Catalogs(ctx context.Context) (*catmodels.Response, error) {
	r := s.uow.Repositories()
	out := &catmodels.Response{OrigenesContacto: origenes}

	deps, err := r.Departamentos().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Departamentos = make([]catmodels.Departamento, 0, len(deps))
	for _, d := range deps {
		out.Departamentos = append(out.Departamentos, catmodels.Departamento{ID: d.ID, Codigo: d.Codigo, Nombre: d.Nombre})
	}

	roles, err := r.Roles().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Roles = make([]catmodels.Rol, 0, len(roles))
	for _, ro := range roles {
		out.Roles = append(out.Roles, catmodels.Rol{ID: ro.ID, Codigo: ro.Codigo, Nombre: ro.Nombre})
	}

	sectores, err := r.Sectores().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Sectores = make([]catmodels.Sector, 0, len(sectores))
	for _, x := range sectores {
		out.Sectores = append(out.Sectores, catmodels.Sector{ID: x.ID, Nombre: x.Nombre, Tipo: string(x.Tipo)})
	}

	props, err := r.Propositos().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.PropositosConsentimiento = make([]catmodels.PropositoConsentimiento, 0, len(props))
	for _, x := range props {
		out.PropositosConsentimiento = append(out.PropositosConsentimiento, catmodels.PropositoConsentimiento{
			ID: x.ID, Codigo: x.Codigo, Descripcion: x.Descripcion, Opcional: x.Opcional,
		})
	}

	return out, nil
}
