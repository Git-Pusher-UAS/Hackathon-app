package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
)

// A21 · expedientes.personas
type PersonaRepository interface {
	Create(ctx context.Context, p *models.Persona) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Persona, error)

	// GetByIDForUpdate bloquea la fila (SELECT ... FOR UPDATE). Se usa antes de asignar
	// orden_persona en consentimiento.decisiones.
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Persona, error)

	Update(ctx context.Context, p *models.Persona, expectedVersion int64) error
}

// IndiceCURPClave identifica un HMAC de CURP bajo una versión de clave concreta.
type IndiceCURPClave struct {
	VersionHMAC int
	ValorHMAC   []byte
}

// A22 · expedientes.indices_curp
type IndiceCURPRepository interface {
	// Create devuelve ErrConflict si ese (version_hmac, valor_hmac) ya existe: duplicado de CURP.
	Create(ctx context.Context, idx *models.IndiceCURP) error

	// BuscarPersona busca por todas las versiones candidatas (coexistencia durante la
	// rotación de la clave HMAC). Devuelve ErrNotFound si no hay coincidencia.
	BuscarPersona(ctx context.Context, candidatos []IndiceCURPClave) (uuid.UUID, error)

	ListByPersona(ctx context.Context, personaID uuid.UUID) ([]models.IndiceCURP, error)

	// ListPersonasSinVersion devuelve IDs de personas que aún no tienen índice con
	// la versión indicada (backfill de rotación), en lotes.
	ListPersonasSinVersion(ctx context.Context, versionHMAC int, limit int) ([]uuid.UUID, error)

	// CountPersonasSinVersion verifica la cobertura total antes de retirar la versión antigua.
	CountPersonasSinVersion(ctx context.Context, versionHMAC int) (int64, error)

	// DeleteVersion retira una versión antigua; solo se permite con cobertura total de la nueva.
	DeleteVersion(ctx context.Context, versionHMAC int) (int64, error)
}

// ExpedienteFilter para GET /staff/records.
type ExpedienteFilter struct {
	// AsignadoA restringe a expedientes con asignación vigente del personal
	// (se refuerza además con RLS).
	AsignadoA      *uuid.UUID
	Estado         *models.EstadoExpediente
	TipoCasoID     *uuid.UUID // del episodio activo
	DepartamentoID *uuid.UUID
	Alias          string
	Actualizado    TimeRange // sobre actualizado_en
}

// A23 · expedientes.expedientes
type ExpedienteRepository interface {
	// Create devuelve ErrConflict si la persona ya tiene expediente o la operación ya se aplicó.
	Create(ctx context.Context, e *models.Expediente) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Expediente, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Expediente, error)
	GetByPersonaID(ctx context.Context, personaID uuid.UUID) (*models.Expediente, error)
	GetByOperacionID(ctx context.Context, operacionID uuid.UUID) (*models.Expediente, error)
	List(ctx context.Context, f ExpedienteFilter, page PageRequest) (Page[models.Expediente], error)

	// Update guarda alias/estado/cerrado_en con control de versión (If-Match).
	// Quien lo invoca debe insertar el HistorialEstado en la misma transacción.
	Update(ctx context.Context, e *models.Expediente, expectedVersion int64) error
}

// A24 · expedientes.episodios
type EpisodioRepository interface {
	Create(ctx context.Context, ep *models.Episodio) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Episodio, error)

	// GetActivo devuelve el episodio con cerrado_en IS NULL; ErrNotFound si no hay.
	GetActivo(ctx context.Context, expedienteID uuid.UUID) (*models.Episodio, error)

	ListByExpediente(ctx context.Context, expedienteID uuid.UUID) ([]models.Episodio, error)

	// SiguienteNumero devuelve max(numero)+1 del expediente (llamar con el expediente bloqueado).
	SiguienteNumero(ctx context.Context, expedienteID uuid.UUID) (int, error)

	Cerrar(ctx context.Context, id uuid.UUID, at time.Time) error

	// ActualizarTipoCaso permite reclasificar el episodio (ver decisión de diseño sobre
	// la categoría inicial "por clasificar").
	ActualizarTipoCaso(ctx context.Context, id, tipoCasoID uuid.UUID) error
}

// A25 · expedientes.historial_estados (inmutable)
type HistorialEstadoRepository interface {
	Append(ctx context.Context, h *models.HistorialEstado) error
	ListByExpediente(ctx context.Context, expedienteID uuid.UUID, page PageRequest) (Page[models.HistorialEstado], error)
}

// A26 · expedientes.asignaciones
type AsignacionRepository interface {
	// Create devuelve ErrConflict si ya existe una asignación vigente igual.
	Create(ctx context.Context, a *models.Asignacion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Asignacion, error)

	// FindVigente busca la asignación con fin IS NULL; ErrNotFound si no hay.
	FindVigente(ctx context.Context, expedienteID, personalID, departamentoID uuid.UUID) (*models.Asignacion, error)

	ListByExpediente(ctx context.Context, expedienteID uuid.UUID, soloVigentes bool) ([]models.Asignacion, error)
	ListVigentesByPersonal(ctx context.Context, personalID uuid.UUID) ([]models.Asignacion, error)

	// TieneAcceso indica si el personal tiene asignación vigente al expediente y, si
	// departamentoID != nil, a ese departamento.
	TieneAcceso(ctx context.Context, personalID, expedienteID uuid.UUID, departamentoID *uuid.UUID) (bool, error)

	// DepartamentosAsignados lista los departamentos del expediente a los que el personal tiene acceso.
	DepartamentosAsignados(ctx context.Context, personalID, expedienteID uuid.UUID) ([]uuid.UUID, error)

	// Cerrar fija fin (revoca el acceso sin borrar historial).
	Cerrar(ctx context.Context, id uuid.UUID, fin time.Time) error
}

// NotaFilter para listar notas de una sección.
type NotaFilter struct {
	EpisodioID *uuid.UUID
	Rango      TimeRange // sobre creado_en
}

// A27 · expedientes.notas — SOLO inserción y lectura. Las correcciones se hacen por adición.
type NotaRepository interface {
	// Create inserta la nota. Las FKs compuestas garantizan que el episodio y la nota
	// corregida pertenecen al mismo expediente y departamento. ErrConflict si operacion_id ya existe.
	Create(ctx context.Context, n *models.Nota) error

	GetByID(ctx context.Context, id uuid.UUID) (*models.Nota, error)
	GetByOperacionID(ctx context.Context, operacionID uuid.UUID) (*models.Nota, error)

	// ListBySeccion lista las notas de un expediente y departamento, por (creado_en, id).
	ListBySeccion(ctx context.Context, expedienteID, departamentoID uuid.UUID, f NotaFilter, page PageRequest) (Page[models.Nota], error)

	// EsPrevia verifica que la nota a corregir existe, es del mismo expediente y
	// departamento y es anterior a la nueva (no auto-referencia ni ciclos).
	EsPrevia(ctx context.Context, corrigeNotaID, expedienteID, departamentoID uuid.UUID) (bool, error)

	// TieneCorreccion indica si ya existe una nota que corrige a notaID.
	TieneCorreccion(ctx context.Context, notaID uuid.UUID) (bool, error)
}

// TimelineFilter para API-023.
type TimelineFilter struct {
	DepartamentoIDs []uuid.UUID // áreas permitidas al actor
	Tipos           []string    // nota | cambio_estado | canalizacion
	Rango           TimeRange
}

// expedientes.v_timeline_autorizado (vista de solo lectura)
type TimelineRepository interface {
	// ListAutorizado aplica el filtro de áreas antes de devolver filas o conteos.
	ListAutorizado(ctx context.Context, expedienteID uuid.UUID, f TimelineFilter, page PageRequest) (Page[models.TimelineItem], error)
}

// A28 · expedientes.tokens_vinculacion
type TokenVinculacionRepository interface {
	Create(ctx context.Context, t *models.TokenVinculacion) error
	GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.TokenVinculacion, error)

	// Consumir marca consumido_en de forma atómica solo si no estaba consumido ni vencido.
	// Devuelve false si el token ya se usó o venció.
	Consumir(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)

	PurgarVencidos(ctx context.Context, before time.Time) (int64, error)
}

// SolicitudVinculacionFilter para API-019 (solo metadatos para resolver).
type SolicitudVinculacionFilter struct {
	Estado        *models.EstadoSolicitudVinculacion
	SolicitanteID *uuid.UUID
}

// A29 · expedientes.solicitudes_vinculacion
type SolicitudVinculacionRepository interface {
	Create(ctx context.Context, s *models.SolicitudVinculacion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SolicitudVinculacion, error)
	GetByTokenID(ctx context.Context, tokenID uuid.UUID) (*models.SolicitudVinculacion, error)
	List(ctx context.Context, f SolicitudVinculacionFilter, page PageRequest) (Page[models.SolicitudVinculacion], error)

	// Resolver cambia estado, resuelta_por, resuelta_en y asignacion_id con control de versión.
	Resolver(ctx context.Context, id uuid.UUID, estado models.EstadoSolicitudVinculacion, resueltaPor uuid.UUID, resueltaEn time.Time, asignacionID *uuid.UUID, expectedVersion int64) error
}
