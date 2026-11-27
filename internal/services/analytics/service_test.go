package analytics

import (
	"testing"
	"time"
)

func TestCorteYRango(t *testing.T) {
	// 26 de noviembre, 02:00 UTC = 25 de noviembre, 19:00 en Los Mochis.
	now := time.Date(2026, 11, 26, 2, 0, 0, 0, time.UTC)
	corte := Corte(now)
	if want := time.Date(2026, 11, 25, 7, 0, 0, 0, time.UTC); !corte.Equal(want) {
		t.Fatalf("corte = %v, quiero %v", corte, want)
	}
	act, ant, err := Rango("6m", corte)
	if err != nil || !act.Hasta.Equal(corte) || !act.Desde.Equal(corte.AddDate(0, -6, 0)) || !ant.Hasta.Equal(act.Desde) {
		t.Fatalf("rango 6m: %+v %+v %v", act, ant, err)
	}
	if _, _, err := Rango("7d", corte); err == nil {
		t.Fatal("period desconocido debía fallar")
	}
}

func TestSupresion(t *testing.T) {
	if Suprimir(4) != nil || *Suprimir(5) != 5 {
		t.Fatal("se suprime todo lo menor a 5")
	}
	if i := indicador("abiertos", "x", 3, 20); !i.Suprimido || i.Valor != nil || i.Detalle != "Suprimido" {
		t.Fatalf("suprimido: %+v", i)
	}
	if i := indicador("abiertos", "x", 128, 116); i.Detalle != "+12 vs. periodo anterior" {
		t.Fatalf("detalle: %q", i.Detalle)
	}
	if i := indicador("abiertos", "x", 10, 2); i.Detalle != "Sin comparación con el periodo anterior" {
		t.Fatalf("anterior suprimido no debe revelar la diferencia: %q", i.Detalle)
	}
}
