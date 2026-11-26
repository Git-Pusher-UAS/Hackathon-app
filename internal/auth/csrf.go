package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/google/uuid"
)

// CSRFResult es la respuesta de GET /auth/csrf-token.
type CSRFResult struct {
	CSRF string `json:"csrf_token"`
}

// RotarCSRF genera un token CSRF nuevo para la sesión y guarda solo su hash.
func (s *Service) RotarCSRF(ctx context.Context, sessionID uuid.UUID) (*CSRFResult, error) {
	csrf, err := randomToken(tamanoTokenCSRF)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(csrf))
	if err := s.uow.Repositories().Sesiones().ActualizarCSRF(ctx, sessionID, hash[:]); err != nil {
		return nil, fmt.Errorf("auth: rotar csrf: %w", err)
	}

	return &CSRFResult{CSRF: csrf}, nil
}

// CSRFValido compara en tiempo constante el token recibido contra el hash guardado.
// Úsalo desde el middleware que protege los POST/PUT/DELETE.
func CSRFValido(hashGuardado []byte, token string) bool {
	if len(hashGuardado) == 0 || token == "" {
		return false
	}
	h := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(hashGuardado, h[:]) == 1
}
