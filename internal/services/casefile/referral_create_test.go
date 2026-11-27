package casefile

import (
	"context"
	"errors"
	"testing"

	"backend-hackathon/internal/httpx"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

func TestCreateReferralValidacionPorTipo(t *testing.T) {
	svc := New(nil, nil, "secret", "curp")
	bad := "x"
	cases := map[string]struct {
		in    casemodels.ReferralCreateRequest
		field string
	}{
		"interna sin destino":     {casemodels.ReferralCreateRequest{Resumen: "Requiere asesoría"}, "destinoId"},
		"externa sin institución": {casemodels.ReferralCreateRequest{Tipo: "externa", Resumen: "Requiere refugio"}, "institucionId"},
		"externa con id inválido": {casemodels.ReferralCreateRequest{Tipo: "externa", InstitucionID: &bad, Resumen: "Requiere refugio"}, "institucionId"},
		"tipo desconocido":        {casemodels.ReferralCreateRequest{Tipo: "otra", Resumen: "x"}, "tipo"},
		"sin resumen":             {casemodels.ReferralCreateRequest{Tipo: "externa", Resumen: ""}, "resumen"},
	}
	for name, tc := range cases {
		var he *httpx.Error
		_, err := svc.CreateReferral(context.Background(), repository.ActorContext{}, uuid.New(), uuid.New(), tc.in)
		if !errors.As(err, &he) || he.Fields[tc.field] == "" {
			t.Errorf("%s: %v", name, err)
		}
	}
}
