// Package catalog define la respuesta de GET /api/catalogs.
package catalog

import "github.com/google/uuid"

type Departamento struct {
	ID     uuid.UUID `json:"id"`
	Codigo string    `json:"codigo"`
	Nombre string    `json:"nombre"`
}

type Rol struct {
	ID     uuid.UUID `json:"id"`
	Codigo string    `json:"codigo"`
	Nombre string    `json:"nombre"`
}

type Sector struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
	Tipo   string    `json:"tipo"`
}

type Opcion struct {
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

type PropositoConsentimiento struct {
	ID          uuid.UUID `json:"id"`
	Codigo      string    `json:"codigo"`
	Descripcion string    `json:"descripcion"`
	Opcional    bool      `json:"opcional"`
}

type Response struct {
	Departamentos            []Departamento            `json:"departamentos"`
	Roles                    []Rol                     `json:"roles"`
	Sectores                 []Sector                  `json:"sectores"`
	OrigenesContacto         []Opcion                  `json:"origenesContacto"`
	PropositosConsentimiento []PropositoConsentimiento `json:"propositosConsentimiento"`
}
