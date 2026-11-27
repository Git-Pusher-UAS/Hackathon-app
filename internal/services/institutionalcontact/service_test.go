package institutionalcontact

import (
	"testing"

	"backend-hackathon/internal/models"
)

func TestOrderPrioritariosPrimero(t *testing.T) {
	got := Order([]models.Institucion{
		{Nombre: "Albergue"},
		{Nombre: "Emergencias", Prioritario: true},
		{Nombre: "Fiscalía"},
		{Nombre: "Línea mujer", Prioritario: true},
	})
	want := []string{"Emergencias", "Línea mujer", "Albergue", "Fiscalía"}
	for i, w := range want {
		if got[i].Nombre != w {
			t.Fatalf("posición %d: %s, quiero %s (%v)", i, got[i].Nombre, w, got)
		}
	}
}
