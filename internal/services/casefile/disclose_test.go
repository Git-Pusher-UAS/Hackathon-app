package casefile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

type discloseState struct {
	persona *models.Persona
}

var discloseStates = map[*dupRepos]*discloseState{}

type disclosePersonas struct {
	repository.PersonaRepository
	st *discloseState
}

func (f disclosePersonas) GetByID(context.Context, uuid.UUID) (*models.Persona, error) {
	return f.st.persona, nil
}

func (f dupAsigs) TieneAcceso(_ context.Context, personal, exp uuid.UUID, _ *uuid.UUID) (bool, error) {
	for _, a := range f.r.asigs {
		if a.ExpedienteID == exp && a.PersonalID == personal {
			return true, nil
		}
	}
	return false, nil
}

func (r *dupRepos) Personas() repository.PersonaRepository {
	return disclosePersonas{st: discloseStates[r]}
}

func TestDiscloseExigeMotivoYAudita(t *testing.T) {
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	exp, persona, actorID := uuid.New(), uuid.New(), uuid.New()
	r := &dupRepos{
		exps:  map[uuid.UUID]*models.Expediente{exp: {ID: exp, PersonaID: persona}},
		asigs: []models.Asignacion{{ExpedienteID: exp, PersonalID: actorID}},
	}
	assignStates[r] = &assignState{claves: map[uuid.UUID]*models.ClaveRegistro{}}
	clave, tel, err := kr.Seal(context.Background(), r.ClavesRegistro(), modulo, "persona", persona, []byte("6680000081"), aad("persona", "telefono", persona))
	if err != nil {
		t.Fatal(err)
	}
	discloseStates[r] = &discloseState{persona: &models.Persona{ID: persona, ClaveID: clave, TelefonoCifrado: tel}}
	svc := New(&dupUOW{r}, kr, "secret", "curp")
	actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}

	var he *httpx.Error
	if _, err := svc.Disclose(context.Background(), actor, exp, casemodels.DisclosureRequest{Campo: "telefono"}); !errors.As(err, &he) || he.Fields["motivo"] != "motivo_required" {
		t.Fatalf("sin motivo: %v", err)
	}
	if _, err := svc.Disclose(context.Background(), actor, exp, casemodels.DisclosureRequest{Campo: "rfc", Motivo: "Contacto de seguimiento"}); !errors.As(err, &he) || he.Fields["campo"] == "" {
		t.Fatalf("campo inválido: %v", err)
	}
	res, err := svc.Disclose(context.Background(), actor, exp, casemodels.DisclosureRequest{Campo: "telefono", Motivo: "Contacto de seguimiento"})
	if err != nil || res.Valor != "6680000081" {
		t.Fatalf("Disclose: %+v %v", res, err)
	}
	if len(r.audited) != 1 || r.audited[0] != "casefile.disclose.telefono" {
		t.Fatalf("auditoría: %v", r.audited)
	}
}

func TestDiscloseDetalleSinValor(t *testing.T) {
	d := "Reveló " + nombresCampo["telefono"] + ". Motivo: Contacto de seguimiento"
	if strings.Contains(d, "668") || !strings.Contains(d, "teléfono") {
		t.Fatalf("detalle: %q", d)
	}
}
