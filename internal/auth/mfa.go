package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	ambitoMFA       = "mfa"
	emisorMFA       = "Sistema de Ayuda Integral"
	duracionDesafio = 5 * time.Minute
)

// MFAChallengeError se devuelve en el login cuando la cuenta requiere MFA.
type MFAChallengeError struct {
	Challenge string
	Next      string
}

func (e *MFAChallengeError) Error() string { return ErrMFARequerido.Error() }
func (e *MFAChallengeError) Unwrap() error { return ErrMFARequerido }

// MFAChallengeResponse documenta la respuesta cuando se requiere MFA.
type MFAChallengeResponse struct {
	ChallengeID    string `json:"challengeId"`
	Next           string `json:"next"`
	ChallengeToken string `json:"challenge_token,omitempty"`
	Error          string `json:"error,omitempty"`
}

// EnrollmentResult es la respuesta de POST /auth/mfa/enrollments.
type EnrollmentResult struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauth_url"`
}

// VerifyMFAInput son los datos de POST /auth/mfa/verify.
// Con PersonalID (Bearer) se confirma el enrolamiento; con Challenge se completa el login.
type VerifyMFAInput struct {
	PersonalID *uuid.UUID
	Challenge  string
	Code       string
	IP         string
	UserAgent  string
	RequestID  uuid.UUID
}

// VerifyMFAResult: Login viene lleno solo cuando se completó un login con challenge.
type VerifyMFAResult struct {
	Activado bool         `json:"activado"`
	Login    *LoginResult `json:"login,omitempty"`
}

// EnrollMFA inicia el enrolamiento TOTP de la cuenta autenticada.
func (s *Service) EnrollMFA(ctx context.Context, personalID, requestID uuid.UUID, ip string) (*EnrollmentResult, error) {
	repos := s.uow.Repositories()

	p, err := repos.Personal().GetByID(ctx, personalID)
	if err != nil {
		return nil, err
	}

	actual, err := repos.MFA().GetByPersonalID(ctx, personalID)
	switch {
	case err == nil && actual.Estado == models.MFAActivo:
		return nil, ErrMFAYaActivo
	case err != nil && !errors.Is(err, repository.ErrNotFound):
		return nil, fmt.Errorf("auth: consultar MFA: %w", err)
	}

	if err == nil {
		if err := repos.MFA().DeletePendiente(ctx, personalID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("auth: eliminar MFA previo: %w", err)
		}
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      emisorMFA,
		AccountName: p.Usuario,
		SecretSize:  32,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, fmt.Errorf("auth: generar secreto totp: %w", err)
	}

	cifrado, err := s.cifrar([]byte(key.Secret()))
	if err != nil {
		return nil, err
	}

	mfa := &models.MFAPersonal{
		PersonalID:     personalID,
		SemillaCifrada: cifrado,
		Estado:         models.MFAPendiente,
		ClaveID:        uuid.Nil,
	}
	if err := repos.MFA().Create(ctx, mfa); err != nil {
		return nil, fmt.Errorf("auth: guardar MFA pendiente: %w", err)
	}

	s.audit(ctx, &personalID, "auth.mfa_enrolamiento", models.ResultadoExito, &personalID, requestID, ip)

	return &EnrollmentResult{Secret: key.Secret(), OtpauthURL: key.URL()}, nil
}

// VerifyMFA valida el código TOTP. Según el modo, activa MFA o completa el login.
func (s *Service) VerifyMFA(ctx context.Context, in VerifyMFAInput) (*VerifyMFAResult, error) {
	repos := s.uow.Repositories()
	now := time.Now()

	var (
		personalID uuid.UUID
		ctxAuth    *models.ContextoAuth
	)
	viaChallenge := strings.TrimSpace(in.Challenge) != ""
	if viaChallenge {
		c, err := s.cargarDesafio(ctx, repos, in.Challenge, now)
		if err != nil {
			return nil, err
		}
		ctxAuth = c
		personalID = *c.PersonalID
	} else if in.PersonalID != nil {
		personalID = *in.PersonalID
	} else {
		return nil, ErrTokenInvalido
	}

	code := strings.TrimSpace(in.Code)
	if code == "" {
		return nil, ErrCodigoMFAInvalido
	}

	// Rate limit propio de MFA (por cuenta + IP).
	clave := s.claveIntento(personalID.String(), in.IP)
	ventana := now.Truncate(ventanaIntentos)

	bloqueado, _, err := repos.ControlIntentos().EstaBloqueado(ctx, ambitoMFA, clave, now)
	if err != nil {
		return nil, fmt.Errorf("auth: comprobar intentos mfa: %w", err)
	}
	if bloqueado {
		return nil, ErrDemasiadosIntentos
	}

	// Cada intento del challenge cuenta, aunque el código sea correcto.
	if viaChallenge {
		intentos, err := repos.ContextosAuth().IncrementarIntentos(ctx, ctxAuth.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDesafioInvalido
			}
			return nil, fmt.Errorf("auth: incrementar intentos mfa: %w", err)
		}
		if intentos >= maxIntentos {
			return nil, ErrDemasiadosIntentos
		}
	}

	mfa, err := repos.MFA().GetByPersonalID(ctx, personalID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrMFANoIniciado
		}
		return nil, fmt.Errorf("auth: consultar MFA: %w", err)
	}

	if viaChallenge {
		if ctxAuth.EnrolamientoID != nil && *ctxAuth.EnrolamientoID != mfa.ID {
			return nil, ErrDesafioInvalido
		}
		if mfa.Estado != models.MFAActivo {
			return nil, ErrMFANoIniciado
		}
	} else {
		if mfa.Estado == models.MFAActivo {
			return nil, ErrMFAYaActivo
		}
		if mfa.Estado != models.MFAPendiente {
			return nil, ErrMFANoIniciado
		}
	}

	secreto, err := s.descifrar(mfa.SemillaCifrada)
	if err != nil {
		return nil, err
	}

	paso, ok := pasoValido(code, string(secreto), now)
	if !ok {
		s.registrarFalloMFA(ctx, repos, clave, ventana, now)
		s.audit(ctx, &personalID, "auth.mfa_fallido", models.ResultadoDenegado, &personalID, in.RequestID, in.IP)
		return nil, ErrCodigoMFAInvalido
	}

	// Confirmación de enrolamiento (Bearer + MFA pendiente).
	if !viaChallenge {
		if err := repos.MFA().Activar(ctx, mfa.ID, now, paso); err != nil {
			return nil, fmt.Errorf("auth: activar MFA: %w", err)
		}
		s.audit(ctx, &personalID, "auth.mfa_activado", models.ResultadoExito, &personalID, in.RequestID, in.IP)
		return &VerifyMFAResult{Activado: true}, nil
	}

	// Login con challenge: consumir el paso y emitir la sesión.
	usado, err := repos.MFA().ConsumirPaso(ctx, mfa.ID, paso)
	if err != nil {
		return nil, fmt.Errorf("auth: consumir paso MFA: %w", err)
	}
	if !usado {
		s.audit(ctx, &personalID, "auth.mfa_fallido", models.ResultadoDenegado, &personalID, in.RequestID, in.IP)
		return nil, ErrCodigoMFAInvalido
	}

	p, err := repos.Personal().GetByID(ctx, personalID)
	if err != nil {
		return nil, err
	}
	if p.Estado != models.PersonalActivo {
		return nil, ErrSuspendido
	}

	consumido, err := repos.ContextosAuth().Consumir(ctx, ctxAuth.ID, now)
	if err != nil {
		return nil, fmt.Errorf("auth: consumir desafío mfa: %w", err)
	}
	if !consumido {
		return nil, ErrDesafioInvalido
	}

	res, err := s.emitirSesion(ctx, repos, p, now, LoginInput{
		IP: in.IP, UserAgent: in.UserAgent, RequestID: in.RequestID,
	})
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &personalID, "auth.mfa_login", models.ResultadoExito, &personalID, in.RequestID, in.IP)
	return &VerifyMFAResult{Login: res}, nil
}

// crearDesafio guarda en contextos_auth un token de un solo uso para completar el login.
func (s *Service) crearDesafio(ctx context.Context, repos repository.Repositories, personalID, mfaID uuid.UUID, now time.Time) (string, error) {
	token, err := randomToken(tamanoTokenCSRF)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(token))
	csrfHash := sha256.Sum256([]byte(token + "|csrf"))
	ctxAuth := &models.ContextoAuth{
		Fase:           models.FaseTOTP,
		PersonalID:     &personalID,
		TokenHash:      hash[:],
		CSRFHash:       csrfHash[:],
		EnrolamientoID: &mfaID,
		VenceEn:        now.Add(duracionDesafio),
	}
	if err := repos.ContextosAuth().Create(ctx, ctxAuth); err != nil {
		return "", fmt.Errorf("auth: crear desafío mfa: %w", err)
	}
	return token, nil
}

// cargarDesafio busca y valida un challenge_token no vencido ni consumido.
func (s *Service) cargarDesafio(ctx context.Context, repos repository.Repositories, token string, now time.Time) (*models.ContextoAuth, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrDesafioInvalido
	}

	hash := sha256.Sum256([]byte(token))
	c, err := repos.ContextosAuth().GetByTokenHash(ctx, hash[:])
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDesafioInvalido
		}
		return nil, fmt.Errorf("auth: buscar desafío mfa: %w", err)
	}
	if c.Fase != models.FaseTOTP || c.PersonalID == nil || c.ConsumidoEn != nil || !c.VenceEn.After(now) {
		return nil, ErrDesafioInvalido
	}
	return c, nil
}

// pasoValido comprueba el código contra la ventana actual y ±1 (30 s) y devuelve
// el paso TOTP que coincidió. La comparación es en tiempo constante.
func pasoValido(code, secreto string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if code == "" || secreto == "" {
		return 0, false
	}

	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	base := now.Unix() / 30

	for _, delta := range []int64{-1, 0, 1} {
		paso := base + delta
		esperado, err := totp.GenerateCodeCustom(secreto, time.Unix(paso*30, 0), opts)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(esperado), []byte(code)) == 1 {
			return paso, true
		}
	}
	return 0, false
}

// registrarFalloMFA incrementa el contador de intentos MFA y bloquea al llegar al máximo.
func (s *Service) registrarFalloMFA(ctx context.Context, repos repository.Repositories, clave []byte, ventana, now time.Time) {
	intentos, err := repos.ControlIntentos().Incrementar(ctx, ambitoMFA, clave, ventana)
	if err == nil && intentos.Contador >= maxIntentos {
		_ = repos.ControlIntentos().Bloquear(ctx, ambitoMFA, clave, ventana, now.Add(duracionBloqueo))
	}
}

// ---------- cifrado del secreto TOTP (AES-256-GCM) ----------

func (s *Service) cifrar(plano []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.mfaKey)
	if err != nil {
		return nil, fmt.Errorf("auth: cifrar: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: cifrar: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("auth: cifrar: %w", err)
	}
	return gcm.Seal(nonce, nonce, plano, nil), nil
}

func (s *Service) descifrar(cifrado []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.mfaKey)
	if err != nil {
		return nil, fmt.Errorf("auth: descifrar: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: descifrar: %w", err)
	}
	if len(cifrado) < gcm.NonceSize() {
		return nil, errors.New("auth: secreto mfa corrupto")
	}
	nonce, datos := cifrado[:gcm.NonceSize()], cifrado[gcm.NonceSize():]
	plano, err := gcm.Open(nil, nonce, datos, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: descifrar: %w", err)
	}
	return plano, nil
}
