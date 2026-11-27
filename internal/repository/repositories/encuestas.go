package repository

import (
	"context"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type EncuestaRepository interface {
	ListConPreguntas(ctx context.Context) ([]models.Encuesta, error)
	GetByIDWithPreguntas(ctx context.Context, id uuid.UUID) (*models.Encuesta, error)
	ListPendingByUsuaria(ctx context.Context, usuariaID uuid.UUID) ([]models.Encuesta, error)
	GetAsignacion(ctx context.Context, encuestaID, usuariaID uuid.UUID) (*models.AsignacionEncuesta, error)
	GetAsignacionByOperationID(ctx context.Context, operationID uuid.UUID) (*models.AsignacionEncuesta, error)
	SaveAsignacion(ctx context.Context, a *models.AsignacionEncuesta) error
	CreateRespuestas(ctx context.Context, rows []models.RespuestaEncuesta) error
	Envios(ctx context.Context, encuestaID uuid.UUID) (int, error)
	Promedio(ctx context.Context, preguntaID uuid.UUID) (float64, int, error)
	ConteoOpciones(ctx context.Context, preguntaID uuid.UUID) (map[string]int, error)
}
