package portal

import "github.com/google/uuid"

type LoginRequest struct {
	Identificador string `json:"identificador"`
}

type VerifyRequest struct {
	ChallengeID uuid.UUID `json:"challengeId"`
	Code        string    `json:"code"`
}

type CreateAccountRequest struct {
	Nombre      string `json:"nombre"`
	CURP        string `json:"curp"`
	Telefono    string `json:"telefono"`
	AceptaAviso bool   `json:"aceptaAviso"`
}

type ChallengeResponse struct {
	ChallengeID     uuid.UUID `json:"challengeId"`
	TelefonoMascara string    `json:"telefonoMascara"`
}

type VerifyResponse struct {
	Alias string `json:"alias"`
}

type Contacto struct {
	ID              uuid.UUID `json:"id"`
	Nombre          string    `json:"nombre"`
	TelefonoMascara string    `json:"telefonoMascara"`
}

type MeResponse struct {
	Alias              string     `json:"alias"`
	TelefonoMascara    string     `json:"telefonoMascara"`
	ContactosConfianza []Contacto `json:"contactosConfianza"`
	SilenciarAvisos    bool       `json:"silenciarAvisos"`
	CompartirUbicacion bool       `json:"compartirUbicacion"`
	NombreDiscreto     bool       `json:"nombreDiscreto"`
}

type PatchMeRequest struct {
	SilenciarAvisos    *bool `json:"silenciarAvisos,omitempty"`
	CompartirUbicacion *bool `json:"compartirUbicacion,omitempty"`
}

type CreateAlertRequest struct {
	Lat         *float64  `json:"lat,omitempty"`
	Lon         *float64  `json:"lon,omitempty"`
	OperationID uuid.UUID `json:"operationId"`
}

type AlertResponse struct {
	ID                  uuid.UUID `json:"id"`
	Estado              string    `json:"estado"`
	UbicacionCompartida bool      `json:"ubicacionCompartida"`
	ContactosAvisados   bool      `json:"contactosAvisados"`
}

type CancelAlertRequest struct {
	Motivo string `json:"motivo"`
}
