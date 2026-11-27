package repository

import (
	"context"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// EncuestaRepository · esquema encuestas. Solo agregados: nunca respuestas
// individuales ni vínculo con la usuaria.
type EncuestaRepository interface {
	// ListConPreguntas devuelve las encuestas con sus preguntas en orden.
	ListConPreguntas(ctx context.Context) ([]models.Encuesta, error)
	// Envios cuenta cuántas veces se respondió la encuesta.
	Envios(ctx context.Context, encuestaID uuid.UUID) (int, error)
	// Promedio de una pregunta de escala y cuántas respuestas lo forman.
	Promedio(ctx context.Context, preguntaID uuid.UUID) (float64, int, error)
	// ConteoOpciones cuenta las respuestas por opción.
	ConteoOpciones(ctx context.Context, preguntaID uuid.UUID) (map[string]int, error)
}
