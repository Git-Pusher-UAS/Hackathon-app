package cv

import "github.com/google/uuid"

type Experiencia struct {
	Empresa     string `json:"empresa"`
	Puesto      string `json:"puesto"`
	Inicio      string `json:"inicio"`
	Fin         string `json:"fin"`
	Descripcion string `json:"descripcion"`
}
type Estudio struct {
	Institucion string `json:"institucion"`
	Grado       string `json:"grado"`
	Inicio      string `json:"inicio"`
	Fin         string `json:"fin"`
}
type CV struct {
	Perfil struct {
		Nombre   string `json:"nombre"`
		Titular  string `json:"titular"`
		Resumen  string `json:"resumen"`
		Contacto string `json:"contacto"`
	} `json:"perfil"`
	Experiencia []Experiencia `json:"experiencia"`
	Estudios    []Estudio     `json:"estudios"`
	Habilidades []string      `json:"habilidades"`
}
type ApplicationView struct {
	ID        uuid.UUID `json:"id"`
	VacanteID uuid.UUID `json:"vacanteId"`
	Puesto    string    `json:"puesto"`
	Empresa   string    `json:"empresa"`
	Estado    string    `json:"estado"`
	CreadoEn  string    `json:"creadoEn"`
}
type InterviewResponseRequest struct {
	Respuesta string `json:"respuesta"`
}
type WithdrawalRequest struct {
	Motivo string `json:"motivo"`
}
