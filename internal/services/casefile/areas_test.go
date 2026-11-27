package casefile

import (
	"context"
	"reflect"
	"testing"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

func TestAreasSinRepetidos(t *testing.T) {
	exp, ts, ps := uuid.New(), uuid.New(), uuid.New()
	r := &dupRepos{
		asigs: []models.Asignacion{{ExpedienteID: exp, DepartamentoID: ts}, {ExpedienteID: exp, DepartamentoID: ps},
			{ExpedienteID: exp, DepartamentoID: ts}, {ExpedienteID: uuid.New(), DepartamentoID: ps}},
		deps: map[uuid.UUID]*models.Departamento{ts: {Codigo: "trabajo_social"}, ps: {Codigo: "psicologia"}},
	}
	cache := map[uuid.UUID]string{}
	got, err := areas(context.Background(), r, exp, cache)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"trabajo_social", "psicologia"}) || len(cache) != 2 {
		t.Fatalf("areas = %v, cache = %v", got, cache)
	}
	vacio, _ := areas(context.Background(), r, uuid.New(), cache)
	if vacio == nil || len(vacio) != 0 {
		t.Fatalf("sin asignaciones debe ser [] y no null: %#v", vacio)
	}
}
