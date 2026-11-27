package gormrepo

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"testing"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// hashSinDetalle es el cálculo anterior a la columna detalle.
func hashSinDetalle(prev []byte, e *models.EventoAuditoria) []byte {
	h := sha256.New()
	h.Write(prev)
	w := func(s string) { _, _ = h.Write([]byte(s)) }
	w(strconv.FormatInt(e.Secuencia, 10) + "|" + e.ActorID.String() + "|" + e.Accion + "|" + e.TipoRecurso + "|" +
		e.RecursoID.String() + "|" + string(e.Resultado) + "|" + e.OcurridoEn.UTC().Format(time.RFC3339Nano) + "|" + "|" + e.RequestID.String())
	return h.Sum(nil)
}

func TestAuditHashDetalle(t *testing.T) {
	actor, rec := uuid.New(), uuid.New()
	e := &models.EventoAuditoria{Secuencia: 7, ActorID: &actor, Accion: "casefile.disclose.telefono", TipoRecurso: "expediente",
		RecursoID: &rec, Resultado: models.ResultadoExito, OcurridoEn: time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC)}
	prev := []byte("anterior")

	if !bytes.Equal(auditHash(prev, e), hashSinDetalle(prev, e)) {
		t.Fatal("sin detalle el hash debe ser el mismo de antes")
	}
	d := "Reveló teléfono. Motivo: Contacto de seguimiento"
	e.Detalle = &d
	if bytes.Equal(auditHash(prev, e), hashSinDetalle(prev, e)) {
		t.Fatal("el detalle debe formar parte del hash")
	}
}
