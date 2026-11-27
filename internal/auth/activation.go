package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const (
	ambitoActivacion = "activacion"
	// DuracionActivacion es la vigencia del código que entrega TI al crear la cuenta.
	DuracionActivacion = 72 * time.Hour
	// Sin 0/O ni 1/I/L, para dictarlo sin errores.
	alfabetoActivacion = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	minPassword        = 12
	maxPassword        = 128
)

// ErrActivacionInvalida es genérico: código inexistente, vencido, usado o de otra cuenta.
var ErrActivacionInvalida = errors.New("código de activación inválido o vencido")

// ErrPasswordInvalida indica una contraseña fuera de la política.
var ErrPasswordInvalida = errors.New("la contraseña debe tener entre 12 y 128 caracteres")

func normalizarCodigo(codigo string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(codigo)))
}

func hashActivacion(secret, codigo string) []byte {
	return security.HMAC(secret, "activacion", normalizarCodigo(codigo))
}

// NuevaActivacion genera un código de un solo uso (XXXX-XXXX) para que la
// persona elija su contraseña. Solo se guarda su HMAC en contextos_auth.
func NuevaActivacion(ctx context.Context, repos repository.Repositories, secret string, personalID uuid.UUID, now time.Time) (string, time.Time, error) {
	var b strings.Builder
	for i := 0; i < 8; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alfabetoActivacion))))
		if err != nil {
			return "", time.Time{}, fmt.Errorf("auth: generar activación: %w", err)
		}
		if i == 4 {
			b.WriteByte('-')
		}
		b.WriteByte(alfabetoActivacion[n.Int64()])
	}
	codigo := b.String()
	relleno := sha256.Sum256([]byte(uuid.NewString()))
	vence := now.Add(DuracionActivacion)
	err := repos.ContextosAuth().Create(ctx, &models.ContextoAuth{
		Fase:       models.FaseActivacion,
		PersonalID: &personalID,
		TokenHash:  hashActivacion(secret, codigo),
		CSRFHash:   relleno[:],
		VenceEn:    vence,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: guardar activación: %w", err)
	}
	return codigo, vence.UTC(), nil
}

// ActivacionInput son los datos de POST /auth/activations.
type ActivacionInput struct {
	Usuario   string
	Codigo    string
	Password  string
	IP        string
	RequestID uuid.UUID
}

// Activar fija la contraseña de una cuenta nueva con su código de activación.
// Después la persona entra con POST /auth/login y configura su segundo factor.
func (s *Service) Activar(ctx context.Context, in ActivacionInput) error {
	if n := utf8.RuneCountInString(in.Password); n < minPassword || n > maxPassword {
		return ErrPasswordInvalida
	}
	usuario := strings.ToLower(strings.TrimSpace(in.Usuario))
	repos := s.uow.Repositories()
	now := time.Now()

	clave := s.claveIntento(usuario, in.IP)
	ventana := now.Truncate(ventanaIntentos)
	bloqueado, _, err := repos.ControlIntentos().EstaBloqueado(ctx, ambitoActivacion, clave, now)
	if err != nil {
		return fmt.Errorf("auth: comprobar intentos: %w", err)
	}
	if bloqueado {
		return ErrDemasiadosIntentos
	}
	fallo := func() error {
		intentos, err := repos.ControlIntentos().Incrementar(ctx, ambitoActivacion, clave, ventana)
		if err == nil && intentos.Contador >= maxIntentos {
			_ = repos.ControlIntentos().Bloquear(ctx, ambitoActivacion, clave, ventana, now.Add(duracionBloqueo))
		}
		s.audit(ctx, nil, "auth.activacion_fallida", models.ResultadoDenegado, nil, in.RequestID, in.IP)
		return ErrActivacionInvalida
	}

	c, err := repos.ContextosAuth().GetByTokenHash(ctx, hashActivacion(s.secret, in.Codigo))
	if errors.Is(err, repository.ErrNotFound) {
		return fallo()
	}
	if err != nil {
		return fmt.Errorf("auth: buscar activación: %w", err)
	}
	if c.Fase != models.FaseActivacion || c.PersonalID == nil || c.ConsumidoEn != nil || !c.VenceEn.After(now) {
		return fallo()
	}
	p, err := repos.Personal().GetByID(ctx, *c.PersonalID)
	if err != nil {
		return err
	}
	if p.Usuario != usuario {
		return fallo()
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return err
	}
	consumido, err := repos.ContextosAuth().Consumir(ctx, c.ID, now)
	if err != nil {
		return fmt.Errorf("auth: consumir activación: %w", err)
	}
	if !consumido {
		return fallo()
	}
	if err := repos.Personal().SetPassword(ctx, p.ID, hash, false); err != nil {
		return fmt.Errorf("auth: fijar contraseña: %w", err)
	}
	s.audit(ctx, &p.ID, "auth.activacion", models.ResultadoExito, &p.ID, in.RequestID, in.IP)
	return nil
}
