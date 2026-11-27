package casefile

import (
	"testing"
	"time"
)

func TestMascaras(t *testing.T) {
	cases := []struct{ got, want string }{
		{maskTail("XEXX010101MSLXXX07", 2), "••••••••••••••••07"},
		{maskTelefono("668 000 0081"), "•• •••• 81"},
		{maskTelefono("12"), "••"},
		{maskNombre("Nombre  Apellido"), "N••••• A•••••••"},
		{maskDomicilio("Col. Centro 123"), "Col. •••••"},
		{maskDomicilio(""), ""},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestEdad(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	if e := edad("1990-05-14", now); e == nil || *e != 36 {
		t.Fatalf("1990-05-14: %v", e)
	}
	if e := edad("1990-12-01", now); e == nil || *e != 35 {
		t.Fatalf("aún no cumple: %v", e)
	}
	if e := edad("no-fecha", now); e != nil {
		t.Fatalf("fecha inválida: %v", *e)
	}
}
