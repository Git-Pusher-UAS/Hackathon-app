// Package casefile contiene los DTO de /api/v1/case-files.
package casefile

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// API-016
type DuplicateCheckRequest struct {
	CURP string `json:"curp"`
}
type DuplicateCheckResponse struct {
	Exists bool `json:"exists"`
	// Solo si exists: el área que lleva el caso y un token de un solo uso
	// (30 min) para POST /case-access-requests sin volver a enviar la CURP.
	Department    *string    `json:"department,omitempty" example:"Trabajo social"`
	LinkToken     *string    `json:"linkToken,omitempty" example:"lt_9f..."`
	LinkExpiresAt *time.Time `json:"linkExpiresAt,omitempty"`
}

type ConsentInput struct {
	Decision     models.DecisionConsentimiento  `json:"decision"`
	Modalidad    models.ModalidadConsentimiento `json:"modalidad"`
	PropositoIDs []string                       `json:"propositoIds"`
	CapturadoEn  time.Time                      `json:"capturadoEn"`
}

// API-017
type CreateRequest struct {
	Alias               string                `json:"alias"`
	Nombre              string                `json:"nombre"`
	CURP                string                `json:"curp"`
	FechaNacimiento     string                `json:"fechaNacimiento"` // YYYY-MM-DD
	Telefono            *string               `json:"telefono"`
	Domicilio           *string               `json:"domicilio"`
	SectorID            string                `json:"sectorId"`
	OrigenContacto      models.OrigenContacto `json:"origenContacto"`
	InstitucionOrigenID *string               `json:"institucionOrigenId"`
	TipoCasoID          *string               `json:"tipoCasoId"`
	Consentimiento      ConsentInput          `json:"consentimiento"`
}

type Summary struct {
	ID             uuid.UUID               `json:"id"`
	Alias          string                  `json:"alias"`
	Estado         models.EstadoExpediente `json:"estado"`
	OrigenContacto models.OrigenContacto   `json:"origenContacto"`
	TipoCasoID     *uuid.UUID              `json:"tipoCasoId"`
	// Areas son los códigos de los departamentos con asignación vigente.
	Areas      []string   `json:"areas"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ClosedAt   *time.Time `json:"closedAt"`
	VersionTag string     `json:"versionTag"`
}

type ListQuery struct {
	Estado models.EstadoExpediente
	Alias  string
	// Desde y Hasta filtran por updatedAt ([Desde, Hasta)).
	Desde, Hasta *time.Time
}

// API-022
type UpdateRequest struct {
	Alias      *string                  `json:"alias"`
	Estado     *models.EstadoExpediente `json:"estado"`
	Motivo     *string                  `json:"motivo"`
	TipoCasoID *string                  `json:"tipoCasoId"`
}

// API-023
type TimelineQuery struct {
	Tipos []string
	Desde *time.Time
	Hasta *time.Time
}
type TimelineItem struct {
	ID             uuid.UUID  `json:"id"`
	Tipo           string     `json:"tipo"`
	DepartamentoID *uuid.UUID `json:"departamentoId"`
	ActorID        uuid.UUID  `json:"actorId"`
	Resumen        string     `json:"resumen"`
	OcurridoEn     time.Time  `json:"ocurridoEn"`
}

// API-028
type DisclosureRequest struct {
	Campo string `json:"campo"` // nombre|curp|fechaNacimiento|telefono|domicilio
}
type DisclosureResponse struct {
	Campo string `json:"campo"`
	Valor string `json:"valor"`
}

// API-024 / API-025
type AssignRequest struct {
	PersonalID     string `json:"personalId"`
	DepartamentoID string `json:"departamentoId"`
	Motivo         string `json:"motivo"`
	// SolicitudID aprueba esa solicitud de acceso en la misma transacción.
	SolicitudID *string `json:"solicitudId,omitempty"`
}
type AssignmentView struct {
	ID             uuid.UUID  `json:"id"`
	PersonalID     uuid.UUID  `json:"personalId"`
	DepartamentoID uuid.UUID  `json:"departamentoId"`
	Inicio         time.Time  `json:"inicio"`
	Fin            *time.Time `json:"fin"`
}

// API-026 / API-027
type NoteCreateRequest struct {
	Texto         string    `json:"texto"`
	OcurridoEn    time.Time `json:"ocurridoEn"`
	CorrigeNotaID *string   `json:"corrigeNotaId"`
}
type NoteView struct {
	ID            uuid.UUID  `json:"id"`
	EpisodioID    uuid.UUID  `json:"episodioId"`
	AutorID       uuid.UUID  `json:"autorId"`
	Texto         string     `json:"texto"`
	OcurridoEn    time.Time  `json:"ocurridoEn"`
	CorrigeNotaID *uuid.UUID `json:"corrigeNotaId"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// API-029 / API-030 / API-031
type ConsentView struct {
	ConsentID   uuid.UUID                     `json:"consentId"`
	PropositoID uuid.UUID                     `json:"propositoId"`
	Decision    models.DecisionConsentimiento `json:"decision"`
	Vigente     bool                          `json:"vigente"`
	Revocado    bool                          `json:"revocado"`
	DecididoEn  time.Time                     `json:"decididoEn"`
	RevocadoEn  *time.Time                    `json:"revocadoEn"`
}
type RevokeConsentRequest struct {
	PropositoIDs []string `json:"propositoIds"`
	Motivo       *string  `json:"motivo"`
}

// API-033
type ReferralCreateRequest struct {
	DestinoID string `json:"destinoId"`
	Resumen   string `json:"resumen"`
}
type ReferralView struct {
	ID           uuid.UUID                 `json:"id"`
	ExpedienteID uuid.UUID                 `json:"expedienteId"`
	OrigenID     uuid.UUID                 `json:"origenId"`
	DestinoID    uuid.UUID                 `json:"destinoId"`
	Estado       models.EstadoCanalizacion `json:"estado"`
	VersionTag   string                    `json:"versionTag"`
	CreatedAt    time.Time                 `json:"createdAt"`
}
