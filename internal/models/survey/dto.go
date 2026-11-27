package survey

import (
	"backend-hackathon/internal/models"
	"github.com/google/uuid"
)

type Pregunta struct {
	ID       string              `json:"id"`
	Texto    string              `json:"texto"`
	Tipo     models.TipoPregunta `json:"tipo"`
	Opciones []string            `json:"opciones,omitempty"`
}

type Encuesta struct {
	ID        uuid.UUID  `json:"id"`
	Titulo    string     `json:"titulo"`
	Minutos   int        `json:"minutos"`
	Preguntas []Pregunta `json:"preguntas"`
}

type SubmitRequest struct {
	Respuestas  map[string]any `json:"respuestas"`
	OperationID uuid.UUID      `json:"operationId"`
}

type Conteo struct {
	Opcion string `json:"opcion"`
	Total  *int   `json:"total"`
}

type ResultadoPregunta struct {
	ID       string              `json:"id"`
	Texto    string              `json:"texto"`
	Tipo     models.TipoPregunta `json:"tipo"`
	Promedio *float64            `json:"promedio,omitempty"`
	Conteo   []Conteo            `json:"conteo,omitempty"`
}

type Resultado struct {
	EncuestaID string              `json:"encuestaId"`
	Titulo     string              `json:"titulo"`
	Respuestas *int                `json:"respuestas"`
	Preguntas  []ResultadoPregunta `json:"preguntas"`
}
