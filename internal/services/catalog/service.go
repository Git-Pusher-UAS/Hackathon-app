// Package catalog implementa GET /catalogs (API-015) y el directorio público
// GET /institutional-contacts (API-039).
package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"backend-hackathon/internal/models"
	catmodels "backend-hackathon/internal/models/catalog"
	contactmodels "backend-hackathon/internal/models/institutionalcontact"
	repository "backend-hackathon/internal/repository/repositories"
)

type Service interface {
	Catalogs(ctx context.Context) (*catmodels.Response, error)
	Contacts(ctx context.Context) (*contactmodels.Response, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

var origenes = []models.OrigenContacto{
	models.OrigenPresencial,
	models.OrigenInstitucion,
	models.OrigenLineaAtencion,
	models.OrigenPatrullaGenero,
	models.OrigenEmergenciaReferida,
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
		out.Departamentos = append(out.Departamentos, catmodels.Departamento{
			Item: catmodels.Item{ID: d.ID, Codigo: d.Codigo, Nombre: d.Nombre}, Asistencial: d.Asistencial})
	}

	roles, err := r.Roles().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Roles = make([]catmodels.Item, 0, len(roles))
	for _, x := range roles {
		out.Roles = append(out.Roles, catmodels.Item{ID: x.ID, Codigo: x.Codigo, Nombre: x.Nombre})
	}

	sectores, err := r.Sectores().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Sectores = make([]catmodels.Sector, 0, len(sectores))
	for _, x := range sectores {
		out.Sectores = append(out.Sectores, catmodels.Sector{
			Item: catmodels.Item{ID: x.ID, Codigo: x.Codigo, Nombre: x.Nombre}, Tipo: x.Tipo})
	}

	tipos, err := r.TiposCaso().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.TiposCaso = make([]catmodels.Item, 0, len(tipos))
	for _, x := range tipos {
		out.TiposCaso = append(out.TiposCaso, catmodels.Item{ID: x.ID, Codigo: x.Codigo, Nombre: x.Nombre})
	}

	props, err := r.Propositos().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Propositos = make([]catmodels.Proposito, 0, len(props))
	for _, x := range props {
		out.Propositos = append(out.Propositos, catmodels.Proposito{ID: x.ID, Codigo: x.Codigo,
			Descripcion: x.Descripcion, Opcional: x.Opcional, Revocable: x.Revocable})
	}

	insts, err := r.Instituciones().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Instituciones = make([]catmodels.Institucion, 0, len(insts))
	for _, x := range insts {
		out.Instituciones = append(out.Instituciones, catmodels.Institucion{ID: x.ID, Nombre: x.Nombre})
	}
	return out, nil
}

func (s *service) Contacts(ctx context.Context) (*contactmodels.Response, error) {
	insts, err := s.uow.Repositories().Instituciones().ListPublicables(ctx)
	if err != nil {
		return nil, err
	}
	out := &contactmodels.Response{Items: make([]contactmodels.Contact, 0, len(insts))}
	h := sha256.New()
	for _, x := range insts {
		out.Items = append(out.Items, contactmodels.Contact{ID: x.ID, Nombre: x.Nombre, Telefono: x.Telefono,
			Horario: x.Horario, RutaAtencion: x.RutaAtencion, VerificadoEn: x.VerificadoEn})
		_, _ = h.Write([]byte(x.ID.String() + ":" + strconv.FormatInt(x.Version, 10) + ";"))
	}
	out.Version = hex.EncodeToString(h.Sum(nil))[:16]
	return out, nil
}
