package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const (
	// IdleTimeout es el máximo de inactividad permitido antes de invalidar una sesión.
	IdleTimeout = 15 * time.Minute

	ambitoLogin     = "login"
	maxIntentos     = 5
	ventanaIntentos = 15 * time.Minute
	duracionBloqueo = 15 * time.Minute
	maxDispositivo  = 100
	tamanoTokenCSRF = 32
)

// Service implementa el login/logout y la consulta del usuario autenticado.
type Service struct {
	uow    repository.UnitOfWork
	secret string
	ttl    time.Duration
	mfaKey []byte
}

// NewService construye el servicio de autenticación.
func NewService(uow repository.UnitOfWork, secret string, ttl time.Duration, mfaKey []byte) (*Service, error) {
	if len(mfaKey) != 32 {
		return nil, errors.New("auth: MFA_ENCRYPTION_KEY debe decodificar a 32 bytes")
	}
	return &Service{uow: uow, secret: secret, ttl: ttl, mfaKey: mfaKey}, nil
}

// LoginInput son los datos de una petición de login.
type LoginInput struct {
	Usuario   string
	Password  string
	IP        string
	UserAgent string
	RequestID uuid.UUID
}

// UsuarioInfo es la representación pública de la cuenta autenticada.
type UsuarioInfo struct {
	ID                     uuid.UUID `json:"id"`
	Usuario                string    `json:"usuario"`
	NombreLaboral          string    `json:"nombre_laboral"`
	Rol                    string    `json:"rol"`
	Permisos               []string  `json:"permisos"`
	RequiereCambioPassword bool      `json:"requiere_cambio_password"`
}

// LoginResult es la respuesta de un login exitoso.
type LoginResult struct {
	Token     string      `json:"token"`
	TokenType string      `json:"token_type"`
	ExpiresIn int64       `json:"expires_in"`
	CSRF      string      `json:"csrf_token"`
	Usuario   UsuarioInfo `json:"usuario"`
}

// Login valida credenciales, aplica rate limit y crea una sesión revocable.
// Si la cuenta tiene MFA activo devuelve *MFAChallengeError con el challenge_token.
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	usuario := strings.ToLower(strings.TrimSpace(in.Usuario))
	if usuario == "" || in.Password == "" {
		return nil, ErrCredenciales
	}

	now := time.Now()
	repos := s.uow.Repositories()
	clave := s.claveIntento(usuario, in.IP)
	ventana := now.Truncate(ventanaIntentos)

	bloqueado, _, err := repos.ControlIntentos().EstaBloqueado(ctx, ambitoLogin, clave, now)
	if err != nil {
		return nil, fmt.Errorf("auth: comprobar intentos: %w", err)
	}
	if bloqueado {
		return nil, ErrDemasiadosIntentos
	}

	p, err := repos.Personal().GetByUsuario(ctx, usuario)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Mismo costo que una verificación real para no filtrar por tiempo.
			DummyVerify(in.Password)
			s.registrarIntento(ctx, repos, clave, ventana, now)
			s.audit(ctx, nil, "auth.login_fallido", models.ResultadoDenegado, nil, in.RequestID, in.IP)
			return nil, ErrCredenciales
		}
		return nil, fmt.Errorf("auth: buscar personal: %w", err)
	}

	if p.Estado != models.PersonalActivo {
		s.audit(ctx, &p.ID, "auth.login_fallido", models.ResultadoDenegado, &p.ID, in.RequestID, in.IP)
		return nil, ErrSuspendido
	}
	if p.BloqueadoHasta != nil && p.BloqueadoHasta.After(now) {
		s.audit(ctx, &p.ID, "auth.login_fallido", models.ResultadoDenegado, &p.ID, in.RequestID, in.IP)
		return nil, ErrBloqueado
	}

	ok, err := VerifyPassword(in.Password, p.PasswordHash)
	if err != nil || !ok {
		s.registrarFallo(ctx, repos, p.ID, clave, ventana, now)
		s.audit(ctx, &p.ID, "auth.login_fallido", models.ResultadoDenegado, &p.ID, in.RequestID, in.IP)
		return nil, ErrCredenciales
	}

	mfa, err := repos.MFA().GetByPersonalID(ctx, p.ID)
	switch {
	case err == nil && mfa.Estado == models.MFAActivo:
		token, err := s.crearDesafio(ctx, repos, p.ID, mfa.ID, now)
		if err != nil {
			return nil, err
		}
		s.audit(ctx, &p.ID, "auth.login_mfa_pendiente", models.ResultadoDenegado, &p.ID, in.RequestID, in.IP)
		return nil, &MFAChallengeError{Challenge: token, Next: "totp"}
	case err != nil && !errors.Is(err, repository.ErrNotFound):
		return nil, fmt.Errorf("auth: consultar MFA: %w", err)
	}

	if err := repos.Personal().ReiniciarFallos(ctx, p.ID); err != nil {
		return nil, fmt.Errorf("auth: reiniciar fallos: %w", err)
	}

	return s.emitirSesion(ctx, repos, p, now, in)
}

// emitirSesion crea la sesión revocable y arma el LoginResult.
// La usan Login (sin MFA) y VerifyMFA (con challenge).
func (s *Service) emitirSesion(
	ctx context.Context,
	repos repository.Repositories,
	p *models.Personal,
	now time.Time,
	in LoginInput,
) (*LoginResult, error) {
	rol, err := s.rolCodigo(ctx, repos, p.RolID)
	if err != nil {
		return nil, err
	}
	permisos, err := repos.Permisos().CodigosEfectivos(ctx, p.ID, now)
	if err != nil {
		return nil, fmt.Errorf("auth: permisos efectivos: %w", err)
	}
	if permisos == nil {
		permisos = []string{}
	}

	sessionID := uuid.New()
	claims := Claims{
		ActorID:     p.ID,
		SessionID:   sessionID,
		Rol:         rol,
		Permisos:    permisos,
		PermVersion: p.VersionPermisos,
	}
	token, expires, err := SignToken(s.secret, s.ttl, p.ID, claims)
	if err != nil {
		return nil, err
	}

	csrf, err := randomToken(tamanoTokenCSRF)
	if err != nil {
		return nil, err
	}

	tokenHash := sha256.Sum256([]byte(token))
	csrfHash := sha256.Sum256([]byte(csrf))
	sesion := &models.SesionPersonal{
		ID:              sessionID,
		TitularID:       p.ID,
		TokenHash:       tokenHash[:],
		CSRFHash:        csrfHash[:],
		UltimaActividad: now,
		VenceAbsoluto:   expires,
		Dispositivo:     recortar(in.UserAgent, maxDispositivo),
		MFAVerificadoEn: now,
		VersionPermisos: p.VersionPermisos,
	}
	if err := repos.Sesiones().Create(ctx, sesion); err != nil {
		return nil, fmt.Errorf("auth: crear sesión: %w", err)
	}

	s.audit(ctx, &p.ID, "auth.login", models.ResultadoExito, &p.ID, in.RequestID, in.IP)

	return &LoginResult{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int64(s.ttl.Seconds()),
		CSRF:      csrf,
		Usuario: UsuarioInfo{
			ID:                     p.ID,
			Usuario:                p.Usuario,
			NombreLaboral:          p.NombreLaboral,
			Rol:                    rol,
			Permisos:               permisos,
			RequiereCambioPassword: p.RequiereCambioPassword,
		},
	}, nil
}

// Me devuelve la información actual de la cuenta autenticada.
func (s *Service) Me(ctx context.Context, personalID uuid.UUID) (*UsuarioInfo, error) {
	repos := s.uow.Repositories()

	p, err := repos.Personal().GetByID(ctx, personalID)
	if err != nil {
		return nil, err
	}
	rol, err := s.rolCodigo(ctx, repos, p.RolID)
	if err != nil {
		return nil, err
	}
	permisos, err := repos.Permisos().CodigosEfectivos(ctx, p.ID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("auth: permisos efectivos: %w", err)
	}
	if permisos == nil {
		permisos = []string{}
	}

	return &UsuarioInfo{
		ID:                     p.ID,
		Usuario:                p.Usuario,
		NombreLaboral:          p.NombreLaboral,
		Rol:                    rol,
		Permisos:               permisos,
		RequiereCambioPassword: p.RequiereCambioPassword,
	}, nil
}

// Logout revoca la sesión actual. Es idempotente.
func (s *Service) Logout(ctx context.Context, personalID, sessionID, requestID uuid.UUID, ip string) error {
	err := s.uow.Repositories().Sesiones().Revocar(ctx, sessionID, time.Now())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("auth: revocar sesión: %w", err)
	}

	s.audit(ctx, &personalID, "auth.logout", models.ResultadoExito, &personalID, requestID, ip)
	return nil
}

func (s *Service) rolCodigo(ctx context.Context, repos repository.Repositories, rolID uuid.UUID) (string, error) {
	rol, err := repos.Roles().GetByID(ctx, rolID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("auth: buscar rol: %w", err)
	}
	return rol.Codigo, nil
}

func (s *Service) registrarFallo(
	ctx context.Context,
	repos repository.Repositories,
	personalID uuid.UUID,
	clave []byte,
	ventana time.Time,
	now time.Time,
) {
	fallos, _, err := repos.Personal().RegistrarFallo(ctx, personalID, nil)
	if err == nil && fallos >= maxIntentos {
		hasta := now.Add(duracionBloqueo)
		_, _, _ = repos.Personal().RegistrarFallo(ctx, personalID, &hasta)
	}

	s.registrarIntento(ctx, repos, clave, ventana, now)
}

// registrarIntento cuenta el fallo por usuario+IP aunque la cuenta no exista.
func (s *Service) registrarIntento(
	ctx context.Context,
	repos repository.Repositories,
	clave []byte,
	ventana time.Time,
	now time.Time,
) {
	intentos, err := repos.ControlIntentos().Incrementar(ctx, ambitoLogin, clave, ventana)
	if err == nil && intentos.Contador >= maxIntentos {
		_ = repos.ControlIntentos().Bloquear(ctx, ambitoLogin, clave, ventana, now.Add(duracionBloqueo))
	}
}

func (s *Service) claveIntento(usuario, ip string) []byte {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(usuario))
	mac.Write([]byte("|"))
	mac.Write([]byte(ip))
	return mac.Sum(nil)
}

func (s *Service) audit(
	ctx context.Context,
	actorID *uuid.UUID,
	accion string,
	resultado models.ResultadoAuditoria,
	recursoID *uuid.UUID,
	requestID uuid.UUID,
	ip string,
) {
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}

	var ipOrigen *string
	if ip != "" {
		v := ip
		ipOrigen = &v
	}

	err := s.uow.DoIndependent(ctx, func(ctx context.Context, r repository.Repositories) error {
		return r.EventosAuditoria().Append(ctx, &models.EventoAuditoria{
			ActorID:     actorID,
			Accion:      accion,
			TipoRecurso: "personal",
			RecursoID:   recursoID,
			Resultado:   resultado,
			OcurridoEn:  time.Now(),
			IPOrigen:    ipOrigen,
			RequestID:   requestID,
		})
	})
	if err != nil {
		log.Printf("auth: no se pudo auditar %s: %v", accion, err)
	}
}

func recortar(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
