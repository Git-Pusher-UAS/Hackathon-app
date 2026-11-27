// Package catalog define los DTOs de GET /catalogs (API-015).
package catalog

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Item es una entrada genérica de catálogo.
type Item struct {
	ID     uuid.UUID `json:"id"`
	Codigo string    `json:"codigo"`
	Nombre string    `json:"nombre"`
}

// Departamento añade si el área es asistencial (lleva notas).
type Departamento struct {
	Item
	Asistencial bool `json:"asistencial"`
}

// Sector es un sector, colonia o sindicatura (sin geometría).
type Sector struct {
	Item
	Tipo models.TipoSector `json:"tipo"`
}

// Proposito es una finalidad del consentimiento.
type Proposito struct {
	ID          uuid.UUID `json:"id"`
	Codigo      string    `json:"codigo"`
	Descripcion string    `json:"descripcion"`
	Opcional    bool      `json:"opcional"`
	Revocable   bool      `json:"revocable"`
}

// Institucion es una institución de origen o destino.
type Institucion struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
}

// Response agrupa los catálogos activos para filtros y formularios.
type Response struct {
	Departamentos    []Departamento          `json:"departamentos"`
	Roles            []Item                  `json:"roles"`
	Sectores         []Sector                `json:"sectores"`
	TiposCaso        []Item                  `json:"tiposCaso"`
	OrigenesContacto []models.OrigenContacto `json:"origenesContacto"`
	Propositos       []Proposito             `json:"propositos"`
	Instituciones    []Institucion           `json:"instituciones"`
}
