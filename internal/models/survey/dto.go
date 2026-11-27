// Package survey define la respuesta de GET /surveys/results (§12.4).
// Solo agregados: null si hay menos de 5 respuestas; las abiertas no se devuelven.
package survey

import "backend-hackathon/internal/models"

// Conteo es el total de respuestas de una opción.
type Conteo struct {
	Opcion string `json:"opcion" example:"Sí"`
	Total  *int   `json:"total" example:"30"`
}

// Pregunta es el agregado de una pregunta. promedio solo en escala, conteo solo en opcion.
type Pregunta struct {
	ID       string              `json:"id" example:"p1"`
	Texto    string              `json:"texto"`
	Tipo     models.TipoPregunta `json:"tipo" enums:"escala,opcion,abierta"`
	Promedio *float64            `json:"promedio,omitempty" example:"4.3"`
	Conteo   []Conteo            `json:"conteo,omitempty"`
}

// Resultado es el agregado de una encuesta.
type Resultado struct {
	EncuestaID string     `json:"encuestaId"`
	Titulo     string     `json:"titulo" example:"¿Cómo te atendimos?"`
	Respuestas *int       `json:"respuestas" example:"37"`
	Preguntas  []Pregunta `json:"preguntas"`
}
