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

// NivelSuprimido es el nivel de un sector con menos de 5 casos.
const NivelSuprimido = "suprimido"

// Celda es el nivel de un AGEB: 1 a 5 por quintiles, o "suprimido".
type Celda struct {
	Ageb  string `json:"ageb" example:"2501000010123"`
	Nivel any    `json:"nivel" swaggertype:"string" example:"3"` // 1..5 o "suprimido"
}

// Heatmap es la respuesta de GET /analytics/heatmap. Nunca lleva conteos.
type Heatmap struct {
	Corte    time.Time `json:"corte"`
	Sectores []Celda   `json:"sectores"`
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
