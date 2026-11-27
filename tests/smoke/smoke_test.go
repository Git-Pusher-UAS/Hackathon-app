//go:build smoke

// Smoke test contra un servidor real en ejecución.
//
//	go test -tags smoke ./tests/smoke/ -v
//	SMOKE_BASE_URL=http://localhost:8080 SMOKE_TOKEN=<jwt> go test -tags smoke ./tests/smoke/ -v
//	go test -tags smoke ./tests/smoke/ -run 'TestSmoke/health' -v   # una sola verificación
package smoke

import (
	"errors"
	"testing"
)

func TestSmoke(t *testing.T) {
	target := TargetFromEnv()

	// Si el servidor no responde, no tiene sentido seguir: un solo fallo claro.
	if _, err := target.do("GET", target.HealthPath, "", nil); err != nil {
		t.Fatalf("el servidor no responde en %s: %v", target.BaseURL, err)
	}

	for _, c := range All() {
		t.Run(c.Name, func(t *testing.T) {
			err := c.Run(target)
			var skip ErrSkip
			switch {
			case errors.As(err, &skip):
				t.Skip(skip.Reason)
			case err != nil:
				t.Error(err)
			}
		})
	}
}
