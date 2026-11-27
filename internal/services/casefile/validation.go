package casefile

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"

	"github.com/google/uuid"
)

var curpRe = regexp.MustCompile(`^[A-Z][AEIOUX][A-Z]{2}\d{6}[HMX][A-Z]{2}[B-DF-HJ-NP-TV-Z]{3}[A-Z0-9]\d$`)

func normalizeCURP(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func vErr(msg string) error { return &ValidationError{Msg: msg} }

func between(s string, min, max int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	return n >= min && n <= max
}

func parseUUID(raw, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, vErr(name + " inv\u00e1lido")
	}
	return id, nil
}

func parseUUIDs(raw []string, min, max int, name string) ([]uuid.UUID, error) {
	if len(raw) < min || len(raw) > max {
		return nil, vErr(name + ": cantidad inv\u00e1lida")
	}
	seen := make(map[uuid.UUID]bool, len(raw))
	out := make([]uuid.UUID, 0, len(raw))
	for _, r := range raw {
		id, err := parseUUID(r, name)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, vErr(name + ": valores repetidos")
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

type parsedCreate struct {
	alias, nombre, curp string
	fecha               time.Time
	sectorID            uuid.UUID
	instID, tipoCasoID  *uuid.UUID
	propositoIDs        []uuid.UUID
}

func validateConsentInput(in casemodels.ConsentInput) ([]uuid.UUID, error) {
	if in.Decision != models.DecisionAceptado && in.Decision != models.DecisionRechazado {
		return nil, vErr("decision debe ser aceptado o rechazado")
	}
	if in.Modalidad != models.ModalidadPresencial && in.Modalidad != models.ModalidadDigital {
		return nil, vErr("modalidad debe ser presencial o digital")
	}
	if in.CapturadoEn.IsZero() {
		return nil, vErr("capturadoEn es obligatorio")
	}
	return parseUUIDs(in.PropositoIDs, 1, 10, "propositoIds")
}

func validateCreate(in casemodels.CreateRequest) (*parsedCreate, error) {
	p := &parsedCreate{alias: strings.TrimSpace(in.Alias), nombre: strings.TrimSpace(in.Nombre), curp: normalizeCURP(in.CURP)}
	if !between(p.alias, 1, 100) {
		return nil, vErr("alias debe tener entre 1 y 100 caracteres")
	}
	if !between(p.nombre, 1, 200) {
		return nil, vErr("nombre debe tener entre 1 y 200 caracteres")
	}
	if !curpRe.MatchString(p.curp) {
		return nil, vErr("curp inv\u00e1lida")
	}
	fecha, err := time.Parse("2006-01-02", in.FechaNacimiento)
	if err != nil || fecha.After(time.Now()) {
		return nil, vErr("fechaNacimiento inv\u00e1lida (YYYY-MM-DD)")
	}
	p.fecha = fecha
	for _, f := range []*string{in.Telefono, in.Domicilio} {
		if f != nil && utf8.RuneCountInString(*f) > 300 {
			return nil, vErr("telefono/domicilio demasiado largos")
		}
	}
	if p.sectorID, err = parseUUID(in.SectorID, "sectorId"); err != nil {
		return nil, err
	}
	switch in.OrigenContacto {
	case models.OrigenPresencial, models.OrigenInstitucion, models.OrigenLineaAtencion,
		models.OrigenPatrullaGenero, models.OrigenEmergenciaReferida:
	default:
		return nil, vErr("origenContacto inv\u00e1lido")
	}
	if in.InstitucionOrigenID != nil {
		id, err := parseUUID(*in.InstitucionOrigenID, "institucionOrigenId")
		if err != nil {
			return nil, err
		}
		p.instID = &id
	}
	if in.TipoCasoID != nil {
		id, err := parseUUID(*in.TipoCasoID, "tipoCasoId")
		if err != nil {
			return nil, err
		}
		p.tipoCasoID = &id
	}
	if in.Consentimiento.Decision != models.DecisionAceptado {
		return nil, vErr("el consentimiento inicial debe ser aceptado")
	}
	if p.propositoIDs, err = validateConsentInput(in.Consentimiento); err != nil {
		return nil, err
	}
	return p, nil
}

func validateUpdate(in casemodels.UpdateRequest) error {
	if in.Alias == nil && in.Estado == nil && in.TipoCasoID == nil {
		return vErr("no hay campos para actualizar")
	}
	if in.Alias != nil && !between(*in.Alias, 1, 100) {
		return vErr("alias debe tener entre 1 y 100 caracteres")
	}
	if in.Estado != nil {
		switch *in.Estado {
		case models.ExpedienteAbierto, models.ExpedienteEnSeguimiento, models.ExpedienteCerrado:
		default:
			return vErr("estado inv\u00e1lido")
		}
	}
	if in.Motivo != nil && utf8.RuneCountInString(*in.Motivo) > 500 {
		return vErr("motivo demasiado largo")
	}
	if in.TipoCasoID != nil {
		if _, err := parseUUID(*in.TipoCasoID, "tipoCasoId"); err != nil {
			return err
		}
	}
	return nil
}

type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Msg
	}
	return e.Msg
}

func transitionAllowed(from, to models.EstadoExpediente) bool {
	switch from {
	case models.ExpedienteAbierto:
		return to == models.ExpedienteEnSeguimiento || to == models.ExpedienteCerrado
	case models.ExpedienteEnSeguimiento:
		return to == models.ExpedienteAbierto || to == models.ExpedienteCerrado
	case models.ExpedienteCerrado:
		return to == models.ExpedienteAbierto || to == models.ExpedienteEnSeguimiento
	}
	return false
}
