// Package user contiene los DTO de las rutas /api/v1/users.
package user

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Actor es la identidad pública de la cuenta autenticada.
type Actor struct {
	ID           uuid.UUID `json:"id"`
	Rol          string    `json:"rol"`
	DepartmentID uuid.UUID `json:"departmentId"`
	DisplayName  string    `json:"displayName"`
	Permissions  []string  `json:"permissions"`
}

// MeResponse es la respuesta de API-006.
type MeResponse struct {
	Actor         Actor     `json:"actor"`
	ExpiresAt     time.Time `json:"expiresAt"`
	IdleExpiresAt time.Time `json:"idleExpiresAt"`
}

// StaffSummary es la representación administrativa de una cuenta.
type StaffSummary struct {
	ID             uuid.UUID             `json:"id"`
	Usuario        string                `json:"usuario"`
	NombreLaboral  string                `json:"nombreLaboral"`
	Rol            string                `json:"rol"`
	DepartamentoID uuid.UUID             `json:"departamentoId"`
	Estado         models.EstadoPersonal `json:"estado"`
	UltimoAcceso   *time.Time            `json:"ultimoAcceso"`
	VersionTag     string                `json:"versionTag"`
}

// ListQuery son los filtros de GET /users.
type ListQuery struct {
	Q      string // usuario o nombre, sin distinguir acentos
	Role   string // código de rol
	Status string // activo | suspendido
	// SoloAtencion limita a personal activo de áreas de atención (asistenciales).
	// Se fuerza para quien lista con users.read_attention y sin admin.
	SoloAtencion bool
}

// SessionSummary es la representación pública de una sesión.
type SessionSummary struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	DeviceLabel    string    `json:"deviceLabel"`
	Current        bool      `json:"current"`
}

// Page es el sobre paginado del contrato.
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// CreateRequest es el cuerpo de API-011.
type CreateRequest struct {
	Usuario        string `json:"usuario"`
	NombreLaboral  string `json:"nombreLaboral"`
	Rol            string `json:"rol"`
	DepartamentoID string `json:"departamentoId"`
	Password       string `json:"password"`
}

// UpdateRequest es el cuerpo de API-012. Los punteros distinguen campo omitido
// de valor cero.
type UpdateRequest struct {
	Rol            *string                `json:"rol"`
	DepartamentoID *string                `json:"departamentoId"`
	Estado         *models.EstadoPersonal `json:"estado"`
}

// RevokeSessionsRequest es el cuerpo de API-013.
type RevokeSessionsRequest struct {
	Motivo string `json:"motivo"`
}

// MFAResetRequest es el cuerpo de API-014.
type MFAResetRequest struct {
	Motivo                 string `json:"motivo"`
	VerificacionPresencial bool   `json:"verificacionPresencial"`
}
