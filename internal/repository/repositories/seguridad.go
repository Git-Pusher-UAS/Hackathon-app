package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"github.com/google/uuid"
	"time"
)

// A19 · seguridad.claves_registro
type ClaveRegistroRepository interface {
	Create(ctx context.Context, c *models.ClaveRegistro) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ClaveRegistro, error)

	// GetByRegistro localiza la clave por (modulo, tipo_registro, registro_id).
	GetByRegistro(ctx context.Context, modulo, tipoRegistro string, registroID uuid.UUID) (*models.ClaveRegistro, error)

	// ListParaRotar devuelve claves con version_kek < versionKEKObjetivo, en lotes.
	ListParaRotar(ctx context.Context, versionKEKObjetivo int, limit int) ([]models.ClaveRegistro, error)

	// ActualizarEnvoltura guarda la DEK reenvuelta y marca rotada_en.
	ActualizarEnvoltura(ctx context.Context, id uuid.UUID, dekEnvuelta, nonce []byte, versionKEK int, rotadaEn time.Time) error
}

// A20 · seguridad.operaciones (idempotencia)
type OperacionRepository interface {
	// Create inserta la operación. Devuelve ErrConflict si ya existe
	// (actor_id, tipo_operacion, operation_id).
	Create(ctx context.Context, op *models.Operacion) error

	GetByID(ctx context.Context, id uuid.UUID) (*models.Operacion, error)

	// Find localiza una operación previa; ErrNotFound si no existe.
	Find(ctx context.Context, actorID uuid.UUID, tipoOperacion string, operationID uuid.UUID) (*models.Operacion, error)

	// FindForUpdate igual que Find pero con SELECT ... FOR UPDATE, para serializar
	// reintentos concurrentes de la misma operación.
	FindForUpdate(ctx context.Context, actorID uuid.UUID, tipoOperacion string, operationID uuid.UUID) (*models.Operacion, error)

	// PurgarResultados anula resultado_cifrado de las operaciones con resultado_vence_en <= before.
	// La marca de aplicada permanece para impedir una segunda escritura.
	PurgarResultados(ctx context.Context, before time.Time) (int64, error)
}
