// Package analytics define las respuestas de /analytics (§13.2 y §13.3).
// Las cifras menores a 5 se suprimen: llegan en null, nunca en 0.
package analytics

import "time"

// SupresionK es el mínimo de casos para publicar una cifra.
const SupresionK = 5

// Indicador es una cifra principal del tablero.
type Indicador struct {
	Clave     string `json:"clave" example:"abiertos"`
	Titulo    string `json:"titulo" example:"Expedientes abiertos"`
	Valor     *int   `json:"valor" example:"128"`
	Detalle   string `json:"detalle" example:"+12 vs. periodo anterior"`
	Suprimido bool   `json:"suprimido"`
}

// Serie es un punto de una serie agregada.
type Serie struct {
	Etiqueta string `json:"etiqueta" example:"2026-06"`
	Valor    *int   `json:"valor" example:"18"`
}

// Summary es la respuesta de GET /analytics/summary.
type Summary struct {
	Corte       time.Time   `json:"corte"`
	Metodologia string      `json:"metodologia"`
	Indicadores []Indicador `json:"indicadores"`
	PorMes      []Serie     `json:"porMes"`
	PorOrigen   []Serie     `json:"porOrigen"`
	PorArea     []Serie     `json:"porArea"`
}
