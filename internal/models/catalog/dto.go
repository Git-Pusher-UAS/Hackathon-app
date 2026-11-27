// Package catalog define la respuesta de GET /catalogs (§4.1 del contrato).
package catalog

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Departamento es un área del personal.
type Departamento struct {
	ID     uuid.UUID `json:"id"`
	Codigo string    `json:"codigo"`
	Nombre string    `json:"nombre"`
}

// Sector es un sector, colonia o sindicatura. Sin geometría.
type Sector struct {
	ID     uuid.UUID         `json:"id"`
	Nombre string            `json:"nombre"`
	Tipo   models.TipoSector `json:"tipo"`
	Ageb   *string           `json:"ageb"`
}

// Opcion es un valor de un vocabulario cerrado con su nombre visible.
type Opcion struct {
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

// Institucion es una institución de origen.
type Institucion struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
}

// InstitucionDestino es una institución a la que se puede canalizar.
type InstitucionDestino struct {
	ID       uuid.UUID `json:"id"`
	Nombre   string    `json:"nombre"`
	Telefono string    `json:"telefono"`
}

// TipoCaso es un tipo de caso.
type TipoCaso struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
}

// Finalidad es una finalidad del consentimiento.
type Finalidad struct {
	ID          uuid.UUID `json:"id"`
	Nombre      string    `json:"nombre"`
	Obligatoria bool      `json:"obligatoria"`
}

// Vacantes agrupa los vocabularios de la bolsa de trabajo.
type Vacantes struct {
	Sector             []string `json:"sector"`
	Horario            []string `json:"horario"`
	TipoEmpleo         []string `json:"tipoEmpleo"`
	Ubicacion          []string `json:"ubicacion"`
	Escolaridad        []string `json:"escolaridad"`
	Habilidades        []string `json:"habilidades"`
	TiempoExperiencia  []string `json:"tiempoExperiencia"`
	Inicio             []string `json:"inicio"`
	MotivosNoSeleccion []string `json:"motivosNoSeleccion"`
}

// Response son los catálogos activos para filtros y formularios.
type Response struct {
	Departamentos             []Departamento       `json:"departamentos"`
	Sectores                  []Sector             `json:"sectores"`
	OrigenesContacto          []Opcion             `json:"origenesContacto"`
	InstitucionesOrigen       []Institucion        `json:"institucionesOrigen"`
	InstitucionesDestino      []InstitucionDestino `json:"institucionesDestino"`
	TiposCaso                 []TipoCaso           `json:"tiposCaso"`
	FinalidadesConsentimiento []Finalidad          `json:"finalidadesConsentimiento"`
	MotivosRevelacion         []string             `json:"motivosRevelacion"`
	Vacantes                  Vacantes             `json:"vacantes"`
}
