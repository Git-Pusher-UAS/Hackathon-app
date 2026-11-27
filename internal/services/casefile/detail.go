package casefile

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const oculto = "•"

// maskTail oculta todo menos los últimos n caracteres alfanuméricos y conserva
// los espacios: "6680000081" -> "••••••••81".
func maskTail(v string, n int) string {
	runes := []rune(strings.TrimSpace(v))
	visibles := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsSpace(runes[i]) {
			continue
		}
		if visibles < n {
			visibles++
			continue
		}
		runes[i] = []rune(oculto)[0]
	}
	return string(runes)
}

// maskTelefono agrupa como "•• •••• 81".
func maskTelefono(v string) string {
	digits := []rune{}
	for _, r := range v {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	if len(digits) < 4 {
		return strings.Repeat(oculto, len(digits))
	}
	return strings.Repeat(oculto, 2) + " " + strings.Repeat(oculto, 4) + " " + string(digits[len(digits)-2:])
}

// maskNombre deja solo la inicial de cada palabra: "Nombre Apellido" -> "N•••• A•••••".
func maskNombre(v string) string {
	parts := strings.Fields(v)
	for i, p := range parts {
		r := []rune(p)
		parts[i] = string(r[0]) + strings.Repeat(oculto, len(r)-1)
	}
	return strings.Join(parts, " ")
}

// maskDomicilio deja solo la primera palabra: "Col. Centro 123" -> "Col. •••••".
func maskDomicilio(v string) string {
	parts := strings.Fields(v)
	if len(parts) == 0 {
		return ""
	}
	return parts[0] + " " + strings.Repeat(oculto, 5)
}

// edad calcula los años cumplidos a partir de AAAA-MM-DD.
func edad(fecha string, now time.Time) *int {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(fecha))
	if err != nil {
		return nil
	}
	years := now.Year() - t.Year()
	if now.Month() < t.Month() || (now.Month() == t.Month() && now.Day() < t.Day()) {
		years--
	}
	return &years
}

// detail arma la respuesta de §5.4: el resumen más sector, edad, áreas y los
// datos sensibles ya enmascarados. Nunca envía la fecha de nacimiento.
func (s *service) detail(ctx context.Context, r repository.Repositories, e *models.Expediente) (*casemodels.Detail, error) {
	tipo, err := activeTipo(ctx, r, e.ID)
	if err != nil {
		return nil, err
	}
	out := &casemodels.Detail{Summary: s.summary(e, tipo)}
	if out.Areas, err = areas(ctx, r, e.ID, map[uuid.UUID]string{}); err != nil {
		return nil, err
	}
	p, err := r.Personas().GetByID(ctx, e.PersonaID)
	if err != nil {
		return nil, err
	}
	if sec, err := r.Sectores().GetByID(ctx, p.SectorID); err == nil {
		out.Sector = sec.Nombre
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	open := func(campo string, payload []byte) (*string, error) {
		if len(payload) == 0 {
			return nil, nil
		}
		v, err := s.open(ctx, r, p.ClaveID, payload, "persona", campo, p.ID)
		if err != nil {
			return nil, err
		}
		return &v, nil
	}
	mask := func(campo string, payload []byte, fn func(string) string) (*string, error) {
		v, err := open(campo, payload)
		if err != nil || v == nil {
			return nil, err
		}
		m := fn(*v)
		return &m, nil
	}
	if out.Ocultos.Nombre, err = mask("nombre", p.NombreCifrado, maskNombre); err != nil {
		return nil, err
	}
	if out.Ocultos.CURP, err = mask("curp", p.CURPCifrada, func(v string) string { return maskTail(v, 2) }); err != nil {
		return nil, err
	}
	if out.Ocultos.Telefono, err = mask("telefono", p.TelefonoCifrado, maskTelefono); err != nil {
		return nil, err
	}
	if out.Ocultos.Domicilio, err = mask("domicilio", p.DomicilioCifrado, maskDomicilio); err != nil {
		return nil, err
	}
	fecha, err := open("fechaNacimiento", p.FechaNacimientoCifrada)
	if err != nil {
		return nil, err
	}
	if fecha != nil {
		out.Edad = edad(*fecha, s.now())
	}
	return out, nil
}
