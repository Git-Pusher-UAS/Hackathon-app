// Package catalog implementa GET /catalogs (§4.1 del contrato).
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

// Orígenes de contacto (vocabulario cerrado del expediente) con su nombre visible.
var origenes = []catmodels.Opcion{
	{Codigo: string(models.OrigenPresencial), Nombre: "Presencial"},
	{Codigo: string(models.OrigenInstitucion), Nombre: "Institución"},
	{Codigo: string(models.OrigenLineaAtencion), Nombre: "Línea de atención"},
	{Codigo: string(models.OrigenPatrullaGenero), Nombre: "Patrulla de género"},
	{Codigo: string(models.OrigenEmergenciaReferida), Nombre: "Emergencia referida"},
}

// MotivosRevelacion son los motivos sugeridos para mostrar un dato oculto.
var MotivosRevelacion = []string{
	"Verificar identidad para canalizar",
	"Contacto de seguimiento",
	"Trámite jurídico en curso",
}

// VocabularioVacantes es el vocabulario de la bolsa de trabajo (§14 a §16).
var VocabularioVacantes = catmodels.Vacantes{
	Sector:      []string{"Servicios", "Comercio", "Agroindustria", "Salud", "Educación"},
	Horario:     []string{"Matutino", "Vespertino", "Mixto"},
	TipoEmpleo:  []string{"Tiempo completo", "Medio tiempo", "Temporal"},
	Ubicacion:   []string{"Los Mochis", "Higuera de Zaragoza", "Topolobampo", "Ahome"},
	Escolaridad: []string{"Sin estudios", "Primaria", "Secundaria", "Preparatoria", "Carrera técnica", "Licenciatura"},
	Habilidades: []string{"Atención a clientes", "Limpieza", "Cocina", "Ventas", "Caja y cobro", "Computadora básica",
		"Cuidado de personas", "Costura", "Campo y empaque", "Manejo de redes sociales", "Organización y archivo", "Trabajo en equipo"},
	TiempoExperiencia:  []string{"Menos de 1 año", "1 a 3 años", "Más de 3 años"},
	Inicio:             []string{"De inmediato", "En 2 semanas", "En 1 mes"},
	MotivosNoSeleccion: []string{"El perfil no coincide", "La vacante se cubrió", "No respondió a la entrevista", "Otro"},
}

func (s *service) Catalogs(ctx context.Context) (*catmodels.Response, error) {
	r := s.uow.Repositories()
	out := &catmodels.Response{OrigenesContacto: origenes, MotivosRevelacion: MotivosRevelacion, Vacantes: VocabularioVacantes}

	deps, err := r.Departamentos().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Departamentos = make([]catmodels.Departamento, 0, len(deps))
	for _, d := range deps {
		out.Departamentos = append(out.Departamentos, catmodels.Departamento{ID: d.ID, Codigo: d.Codigo, Nombre: d.Nombre})
	}

	sectores, err := r.Sectores().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.Sectores = make([]catmodels.Sector, 0, len(sectores))
	for _, x := range sectores {
		out.Sectores = append(out.Sectores, catmodels.Sector{ID: x.ID, Nombre: x.Nombre, Tipo: x.Tipo, Ageb: x.Ageb})
	}

	// El modelo no distingue instituciones de origen y de destino: ambas listas
	// salen de las instituciones activas.
	insts, err := r.Instituciones().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.InstitucionesOrigen = make([]catmodels.Institucion, 0, len(insts))
	out.InstitucionesDestino = make([]catmodels.InstitucionDestino, 0, len(insts))
	for _, x := range insts {
		out.InstitucionesOrigen = append(out.InstitucionesOrigen, catmodels.Institucion{ID: x.ID, Nombre: x.Nombre})
		out.InstitucionesDestino = append(out.InstitucionesDestino, catmodels.InstitucionDestino{ID: x.ID, Nombre: x.Nombre, Telefono: x.Telefono})
	}

	tipos, err := r.TiposCaso().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.TiposCaso = make([]catmodels.TipoCaso, 0, len(tipos))
	for _, x := range tipos {
		out.TiposCaso = append(out.TiposCaso, catmodels.TipoCaso{ID: x.ID, Nombre: x.Nombre})
	}

	props, err := r.Propositos().List(ctx, true)
	if err != nil {
		return nil, err
	}
	out.FinalidadesConsentimiento = make([]catmodels.Finalidad, 0, len(props))
	for _, x := range props {
		out.FinalidadesConsentimiento = append(out.FinalidadesConsentimiento,
			catmodels.Finalidad{ID: x.ID, Nombre: x.Descripcion, Obligatoria: !x.Opcional})
	}
	return out, nil
}
