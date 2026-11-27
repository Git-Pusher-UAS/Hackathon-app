package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ---- Errores centinela (las implementaciones deben traducir los errores de GORM/pgx) ----

var (
	ErrNotFound        = errors.New("repository: registro no encontrado")
	ErrConflict        = errors.New("repository: conflicto de unicidad o de estado")
	ErrVersionMismatch = errors.New("repository: la versión no coincide (If-Match)")
	ErrImmutable       = errors.New("repository: el registro es inmutable")
	ErrForeignKey      = errors.New("repository: violación de llave foránea o de integridad referencial")
)

// ---- Paginación por cursor ----

type PageRequest struct {
	Limit  int    // tamaño de página; la implementación aplica un máximo
	Cursor string // cursor opaco; vacío = primera página
}

type Page[T any] struct {
	Items      []T
	NextCursor string // vacío = no hay más páginas
}

// TimeRange es un intervalo semiabierto [Desde, Hasta). Un cero significa sin límite.
type TimeRange struct {
	Desde time.Time
	Hasta time.Time
}

// ---- Contexto de actor y Unit of Work ----

// ActorContext se propaga a la sesión de BD para RLS y auditoría.
type ActorContext struct {
	ActorID   uuid.UUID
	RequestID uuid.UUID
	IPOrigen  *string
}

// UnitOfWork delimita una transacción. Si fn devuelve error se hace rollback.
type UnitOfWork interface {
	// Do ejecuta fn en una transacción con el actor establecido como contexto local.
	Do(ctx context.Context, actor ActorContext, fn func(ctx context.Context, r Repositories) error) error

	// DoSystem ejecuta fn en una transacción sin actor de personal (login, bootstrap,
	// directorio público, tareas internas). Usa el actor de tipo sistema.
	DoSystem(ctx context.Context, fn func(ctx context.Context, r Repositories) error) error

	// DoIndependent abre una transacción separada de cualquier otra en curso. Sirve para
	// insertar la auditoría de una denegación aunque la transacción de negocio haga rollback.
	DoIndependent(ctx context.Context, fn func(ctx context.Context, r Repositories) error) error

	// Repositories devuelve repositorios sin transacción explícita (lecturas simples).
	Repositories() Repositories
}

// Repositories agrupa todos los repositorios ligados a la misma conexión/transacción.
type Repositories interface {
	// catalogos
	Departamentos() DepartamentoRepository
	Roles() RolRepository
	Permisos() PermisoRepository
	RolPermisos() RolPermisoRepository
	Sectores() SectorRepository
	TiposCaso() TipoCasoRepository
	Instituciones() InstitucionRepository
	Propositos() PropositoRepository
	AvisosPrivacidad() AvisoPrivacidadRepository
	AvisoPropositos() AvisoPropositoRepository

	// identidad
	Actores() ActorRepository
	Personal() PersonalRepository
	Delegaciones() DelegacionRepository
	MFA() MFAPersonalRepository
	Sesiones() SesionPersonalRepository
	ContextosAuth() ContextoAuthRepository
	ControlIntentos() ControlIntentosRepository
	RecuperacionesMFA() RecuperacionMFARepository

	// seguridad
	ClavesRegistro() ClaveRegistroRepository
	Operaciones() OperacionRepository

	// expedientes
	Personas() PersonaRepository
	IndicesCURP() IndiceCURPRepository
	Expedientes() ExpedienteRepository
	Episodios() EpisodioRepository
	HistorialEstados() HistorialEstadoRepository
	Asignaciones() AsignacionRepository
	Notas() NotaRepository
	TokensVinculacion() TokenVinculacionRepository
	SolicitudesVinculacion() SolicitudVinculacionRepository
	Timeline() TimelineRepository

	// consentimiento
	Decisiones() DecisionRepository
	DecisionPropositos() DecisionPropositoRepository
	Revocaciones() RevocacionRepository
	RevocacionPropositos() RevocacionPropositoRepository
	EstadoPropositos() EstadoPropositoRepository

	// canalizacion
	Canalizaciones() CanalizacionRepository
	HistorialCanalizaciones() HistorialCanalizacionRepository

	// auditoria
	EventosAuditoria() EventoAuditoriaRepository
	AlertasAuditoria() AlertaAuditoriaRepository

	// ayuda
	EventosAyuda() EventoLocalAyudaRepository

	// analitica (vistas)
	Resumen() ResumenRepository
	Mapa() MapaRepository
	Indicadores() IndicadoresRepository

	// capacitacion
	Cursos() CursoRepository
	Inscripciones() InscripcionRepository
}
