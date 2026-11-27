package analytics

import "time"

const SupresionK = 5

type Indicador struct {
	Clave     string `json:"clave"`
	Titulo    string `json:"titulo"`
	Valor     *int   `json:"valor"`
	Detalle   string `json:"detalle,omitempty"`
	Suprimido bool   `json:"suprimido"`
}

type Serie struct {
	Etiqueta  string `json:"etiqueta"`
	Valor     *int   `json:"valor"`
	Suprimido bool   `json:"suprimido"`
}

type Summary struct {
	Corte       time.Time   `json:"corte"`
	Metodologia string      `json:"metodologia"`
	Indicadores []Indicador `json:"indicadores"`
	PorMes      []Serie     `json:"porMes"`
	PorOrigen   []Serie     `json:"porOrigen"`
	PorArea     []Serie     `json:"porArea"`
}

type HeatmapCell struct {
	SectorID   string `json:"sectorId"`
	Nivel      int    `json:"nivel,omitempty"`
	Count      *int   `json:"count"`
	Suppressed bool   `json:"suppressed"`
}
