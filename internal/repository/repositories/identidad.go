package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
)

// A11 · identidad.actores
type ActorRepository interface {
	Create(ctx context.Context, actor *models.Actor) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Actor, error)

	// GetSistema devuelve el actor de tipo sistema al que se atribuyen los procesos internos.
	GetSistema(ctx context.Context) (*models.Actor, error)

	SetActivo(ctx context.Context, id uuid.UUID, activo bool) error
}

// PersonalFilter para GET /staff/users.
type PersonalFilter struct {
	Estado         *models.EstadoPersonal
	RolID          *uuid.UUID
	DepartamentoID *uuid.UUID
	Busqueda       string // coincidencia por usuario o nombre laboral
}

// A12 · identidad.personal
type PersonalRepository interface {
	// Create inserta la cuenta; el ID debe ser el de un actor tipo personal ya creado
	// (en la misma transacción).
	Create(ctx context.Context, p *models.Personal) error

	GetByID(ctx context.Context, id uuid.UUID) (*models.Personal, error)

	// GetByUsuario busca por lower(usuario); el valor llega ya normalizado.
	GetByUsuario(ctx context.Context, usuario string) (*models.Personal, error)

	List(ctx context.Context, f PersonalFilter, page PageRequest) (Page[models.Personal], error)

	// Update guarda cambios con control de versión.
	Update(ctx context.Context, p *models.Personal, expectedVersion int64) error

	// IncrementarVersionPermisos invalida sesiones previas por cambio de rol/permisos.
	// Devuelve la nueva version_permisos.
	IncrementarVersionPermisos(ctx context.Context, id uuid.UUID) (int64, error)

	// RegistrarFallo incrementa fallos_consecutivos atómicamente y, si procede,
	// fija bloqueado_hasta. Devuelve el estado resultante.
	RegistrarFallo(ctx context.Context, id uuid.UUID, bloquearHasta *time.Time) (fallos int, bloqueadoHasta *time.Time, err error)

	// ReiniciarFallos pone fallos_consecutivos = 0 y bloqueado_hasta = NULL tras un login exitoso.
	ReiniciarFallos(ctx context.Context, id uuid.UUID) error

	SetPassword(ctx context.Context, id uuid.UUID, hash string, requiereCambio bool) error

	// LockAdministradoresActivos bloquea las filas de administradores activos (SELECT ... FOR UPDATE)
	// y devuelve cuántos son. Serializa cambios que podrían dejar al sistema sin administrador.
	LockAdministradoresActivos(ctx context.Context) (int, error)

	// LockActivosConPermiso bloquea las filas de cuentas activas cuyo rol incluye
	// el permiso indicado y devuelve sus ids. Serializa cambios que podrían dejar
	// al sistema sin ningún administrador activo.
	LockActivosConPermiso(ctx context.Context, codigoPermiso string) ([]uuid.UUID, error)
}

// A13 · identidad.delegaciones
type DelegacionRepository interface {
	Create(ctx context.Context, d *models.Delegacion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Delegacion, error)

	// ListVigentes devuelve delegaciones no revocadas con inicio <= at < fin.
	ListVigentes(ctx context.Context, personalID uuid.UUID, at time.Time) ([]models.Delegacion, error)

	// ExisteSolapamiento comprueba que no haya otra delegación de la misma capacidad
	// al mismo personal en [inicio, fin).
	ExisteSolapamiento(ctx context.Context, personalID, permisoID uuid.UUID, inicio, fin time.Time) (bool, error)

	Revocar(ctx context.Context, id uuid.UUID, at time.Time) error
}

// A14 · identidad.mfa_personal
type MFAPersonalRepository interface {
	Create(ctx context.Context, m *models.MFAPersonal) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.MFAPersonal, error)
	GetByPersonalID(ctx context.Context, personalID uuid.UUID) (*models.MFAPersonal, error)

	// Activar confirma el enrolamiento: estado = activo, confirmado_en y ultimo_paso_usado.
	Activar(ctx context.Context, id uuid.UUID, confirmadoEn time.Time, paso int64) error

	// ConsumirPaso hace compare-and-set atómico: solo avanza si paso > ultimo_paso_usado.
	// Devuelve false si el paso ya se usó (reutilización de TOTP).
	ConsumirPaso(ctx context.Context, id uuid.UUID, paso int64) (bool, error)

	// Invalidar marca estado = invalidado (reset de MFA).
	Invalidar(ctx context.Context, personalID uuid.UUID) error

	// DeletePendiente elimina un enrolamiento no activo (pendiente o invalidado)
	// para permitir un nuevo enrolamiento.
	DeletePendiente(ctx context.Context, personalID uuid.UUID) error
}

// A15 · identidad.sesiones_personal
type SesionPersonalRepository interface {
	Create(ctx context.Context, s *models.SesionPersonal) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SesionPersonal, error)
	GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.SesionPersonal, error)

	// ListByTitular lista las sesiones del titular (API-008) con paginación por
	// cursor; soloActivas excluye revocadas y vencidas.
	ListByTitular(ctx context.Context, titularID uuid.UUID, soloActivas bool, now time.Time, page PageRequest) (Page[models.SesionPersonal], error)

	// Touch actualiza ultima_actividad.
	Touch(ctx context.Context, id uuid.UUID, now time.Time) error

	// ActualizarCSRF rota el hash CSRF de una sesión activa.
	ActualizarCSRF(ctx context.Context, id uuid.UUID, csrfHash []byte) error

	Revocar(ctx context.Context, id uuid.UUID, at time.Time) error

	// RevocarTodasDeTitular revoca todas las sesiones no revocadas del titular y
	// devuelve cuántas fueron (suspensión, cambio de rol, reset MFA).
	RevocarTodasDeTitular(ctx context.Context, titularID uuid.UUID, at time.Time) (int64, error)

	// PurgarAnteriores elimina sesiones revocadas/vencidas anteriores a before (retención).
	PurgarAnteriores(ctx context.Context, before time.Time) (int64, error)
}

// A16 · identidad.contextos_auth
type ContextoAuthRepository interface {
	Create(ctx context.Context, c *models.ContextoAuth) error
	GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.ContextoAuth, error)

	// IncrementarIntentos suma 1 atómicamente y devuelve el total (máx. 5).
	IncrementarIntentos(ctx context.Context, id uuid.UUID) (int, error)

	// VincularEnrolamiento asocia el contexto al registro MFA pendiente.
	VincularEnrolamiento(ctx context.Context, id, mfaID uuid.UUID) error

	// Consumir marca consumido_en; devuelve false si ya estaba consumido.
	Consumir(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)

	PurgarVencidos(ctx context.Context, before time.Time) (int64, error)
}

// A17 · identidad.control_intentos
type ControlIntentosRepository interface {
	// Incrementar hace UPSERT atómico sobre (ambito, clave_hmac, ventana_inicio)
	// y devuelve la fila resultante.
	Incrementar(ctx context.Context, ambito string, claveHMAC []byte, ventanaInicio time.Time) (*models.ControlIntentos, error)

	// Bloquear fija bloqueado_hasta en la ventana indicada.
	Bloquear(ctx context.Context, ambito string, claveHMAC []byte, ventanaInicio time.Time, hasta time.Time) error

	// EstaBloqueado informa si existe un bloqueo vigente para el ámbito y la clave.
	EstaBloqueado(ctx context.Context, ambito string, claveHMAC []byte, now time.Time) (bool, *time.Time, error)

	PurgarAnteriores(ctx context.Context, before time.Time) (int64, error)
}

// A18 · identidad.recuperaciones_mfa
type RecuperacionMFARepository interface {
	Create(ctx context.Context, r *models.RecuperacionMFA) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.RecuperacionMFA, error)

	// GetPendiente devuelve la recuperación sin completar de la cuenta, si existe.
	GetPendiente(ctx context.Context, personalID uuid.UUID) (*models.RecuperacionMFA, error)

	Completar(ctx context.Context, id uuid.UUID, at time.Time) error
}
