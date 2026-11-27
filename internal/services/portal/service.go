package portal

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	pmodels "backend-hackathon/internal/models/portal"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidCode   = errors.New("invalid code")
	ErrChallengeGone = errors.New("challenge expired")
	ErrConflict      = errors.New("conflict")
)

type LoginResult struct {
	Challenge pmodels.ChallengeResponse
}

type VerifyResult struct {
	Response  pmodels.VerifyResponse
	Token     string
	CSRFToken string
	Session   *models.PortalSesion
}

type Service interface {
	Login(ctx context.Context, identificador string) (*pmodels.ChallengeResponse, error)
	Verify(ctx context.Context, challengeID uuid.UUID, code string) (*VerifyResult, error)
	CreateAccount(ctx context.Context, in pmodels.CreateAccountRequest) (*pmodels.ChallengeResponse, error)
	Me(ctx context.Context, cuentaID uuid.UUID) (*pmodels.MeResponse, error)
	PatchMe(ctx context.Context, cuentaID uuid.UUID, in pmodels.PatchMeRequest) (*pmodels.MeResponse, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	CreateAlert(ctx context.Context, cuentaID uuid.UUID, in pmodels.CreateAlertRequest) (*pmodels.AlertResponse, error)
	NotifyTrustedContacts(ctx context.Context, cuentaID uuid.UUID, alertID uuid.UUID) (*pmodels.AlertResponse, error)
	CancelAlert(ctx context.Context, cuentaID uuid.UUID, alertID uuid.UUID, motivo string) (*pmodels.AlertResponse, error)
}

type service struct {
	uow       repository.UnitOfWork
	now       func() time.Time
	devMode   bool
	logf      func(string, ...any)
	otpSecret []byte
}

func New(uow repository.UnitOfWork, devMode bool, otpSecret string) Service {
	secret := []byte(strings.TrimSpace(otpSecret))
	if len(secret) == 0 {
		secret = []byte("dev-otp-secret")
	}
	return &service{uow: uow, now: time.Now, devMode: devMode, logf: log.Printf, otpSecret: secret}
}

func normalizeCURP(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
func normalizeName(s string) string { return strings.Join(strings.Fields(strings.TrimSpace(s)), " ") }

func maskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if len(p) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(p)-4) + p[len(p)-4:]
}

func randomDigits(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		b[i] = digits[x.Int64()]
	}
	return string(b), nil
}

func hashToken(raw string) []byte { s := sha256.Sum256([]byte(raw)); return s[:] }
func (s *service) otpHash(challengeID uuid.UUID, code string) []byte {
	m := hmac.New(sha256.New, s.otpSecret)
	m.Write([]byte(challengeID.String()))
	m.Write([]byte(":"))
	m.Write([]byte(strings.TrimSpace(code)))
	return m.Sum(nil)
}
func (s *service) fakeChallenge() *pmodels.ChallengeResponse {
	return &pmodels.ChallengeResponse{ChallengeID: uuid.New(), TelefonoMascara: "****0000"}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func newAlias() (string, error) {
	code, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	return "Usuaria-" + code, nil
}

func (s *service) createChallenge(ctx context.Context, r repository.Repositories, cuentaID uuid.UUID, telefono string) (*pmodels.ChallengeResponse, error) {
	code, err := randomDigits(6)
	if err != nil {
		return nil, err
	}
	d := &models.PortalDesafioOTP{ID: uuid.New(), CuentaID: cuentaID, VenceEn: s.now().Add(10 * time.Minute), CreadoEn: s.now()}
	d.CodigoHash = s.otpHash(d.ID, code)
	if err := r.PortalDesafios().Create(ctx, d); err != nil {
		return nil, err
	}
	if s.devMode {
		s.logf("[DEV OTP] portal challenge=%s telefono=%s code=%s", d.ID, maskPhone(telefono), code)
	}
	return &pmodels.ChallengeResponse{ChallengeID: d.ID, TelefonoMascara: maskPhone(telefono)}, nil
}

func (s *service) Login(ctx context.Context, identificador string) (*pmodels.ChallengeResponse, error) {
	id := strings.TrimSpace(identificador)
	if id == "" {
		return nil, httpx.Invalid("identificador requerido", map[string]string{"identificador": "requerido"})
	}
	var out *pmodels.ChallengeResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		curp := normalizeCURP(id)
		cuenta, err := r.PortalCuentas().GetByCURPNormalizada(ctx, curp)
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return err
			}
			cuenta, err = r.PortalCuentas().FindByNombre(ctx, normalizeName(id))
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					out = s.fakeChallenge()
					return nil
				}
				return err
			}
		}
		out, err = s.createChallenge(ctx, r, cuenta.ID, cuenta.Telefono)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) CreateAccount(ctx context.Context, in pmodels.CreateAccountRequest) (*pmodels.ChallengeResponse, error) {
	nombre := normalizeName(in.Nombre)
	curp := normalizeCURP(in.CURP)
	tel := strings.TrimSpace(in.Telefono)
	fields := map[string]string{}
	if nombre == "" {
		fields["nombre"] = "requerido"
	}
	if len(curp) != 18 {
		fields["curp"] = "formato_invalido"
	}
	if tel == "" {
		fields["telefono"] = "requerido"
	}
	if !in.AceptaAviso {
		fields["aceptaAviso"] = "requerido"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("solicitud invalida", fields)
	}
	var out *pmodels.ChallengeResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		_, err := r.PortalCuentas().GetByCURPNormalizada(ctx, curp)
		if err == nil {
			return ErrConflict
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		alias, err := newAlias()
		if err != nil {
			return err
		}
		now := s.now()
		cuenta := &models.PortalCuenta{ID: uuid.New(), Alias: alias, Nombre: nombre, CURPNormalizada: curp, Telefono: tel, AceptaAviso: in.AceptaAviso, CreadoEn: now, ActualizadoEn: now}
		if err := r.PortalCuentas().Create(ctx, cuenta); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrConflict
			}
			return err
		}
		out, err = s.createChallenge(ctx, r, cuenta.ID, cuenta.Telefono)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) Verify(ctx context.Context, challengeID uuid.UUID, code string) (*VerifyResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, httpx.Invalid("code requerido", map[string]string{"code": "requerido"})
	}
	var out *VerifyResult
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		d, err := r.PortalDesafios().GetByID(ctx, challengeID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrChallengeGone
			}
			return err
		}
		now := s.now()
		if d.ConsumidoEn != nil || !d.VenceEn.After(now) || d.Intentos >= 5 {
			return ErrChallengeGone
		}
		expected := s.otpHash(d.ID, code)
		if subtle.ConstantTimeCompare(expected, d.CodigoHash) != 1 {
			d.Intentos++
			_ = r.PortalDesafios().Save(ctx, d)
			return ErrInvalidCode
		}
		d.ConsumidoEn = &now
		if err := r.PortalDesafios().Save(ctx, d); err != nil {
			return err
		}
		cuenta, err := r.PortalCuentas().GetByID(ctx, d.CuentaID)
		if err != nil {
			return err
		}
		token, err := randomToken()
		if err != nil {
			return err
		}
		csrf, err := randomToken()
		if err != nil {
			return err
		}
		ses := &models.PortalSesion{ID: uuid.New(), CuentaID: cuenta.ID, TokenHash: hashToken(token), CSRFHash: hashToken(csrf), UltimaActividad: now, VenceAbsoluto: now.Add(24 * time.Hour), CreadoEn: now}
		if err := r.PortalSesiones().Create(ctx, ses); err != nil {
			return err
		}
		out = &VerifyResult{Response: pmodels.VerifyResponse{Alias: cuenta.Alias}, Token: ses.ID.String() + "." + token, CSRFToken: csrf, Session: ses}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) Me(ctx context.Context, cuentaID uuid.UUID) (*pmodels.MeResponse, error) {
	var out *pmodels.MeResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		cuenta, err := r.PortalCuentas().GetByID(ctx, cuentaID)
		if err != nil {
			return err
		}
		contactos, err := r.PortalContactos().ListByCuentaID(ctx, cuentaID)
		if err != nil {
			return err
		}
		items := make([]pmodels.Contacto, 0, len(contactos))
		for _, c := range contactos {
			items = append(items, pmodels.Contacto{ID: c.ID, Nombre: c.Nombre, TelefonoMascara: maskPhone(c.Telefono)})
		}
		out = &pmodels.MeResponse{Alias: cuenta.Alias, TelefonoMascara: maskPhone(cuenta.Telefono), ContactosConfianza: items, SilenciarAvisos: cuenta.SilenciarAvisos, CompartirUbicacion: cuenta.CompartirUbicacion, NombreDiscreto: cuenta.NombreDiscreto}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) PatchMe(ctx context.Context, cuentaID uuid.UUID, in pmodels.PatchMeRequest) (*pmodels.MeResponse, error) {
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		cuenta, err := r.PortalCuentas().GetByID(ctx, cuentaID)
		if err != nil {
			return err
		}
		if in.SilenciarAvisos != nil {
			cuenta.SilenciarAvisos = *in.SilenciarAvisos
		}
		if in.CompartirUbicacion != nil {
			cuenta.CompartirUbicacion = *in.CompartirUbicacion
		}
		cuenta.ActualizadoEn = s.now()
		return r.PortalCuentas().Update(ctx, cuenta)
	})
	if err != nil {
		return nil, err
	}
	return s.Me(ctx, cuentaID)
}

func alertResponse(a *models.PortalAlertaAyuda) *pmodels.AlertResponse {
	return &pmodels.AlertResponse{ID: a.ID, Estado: string(a.Estado), UbicacionCompartida: a.UbicacionCompartida, ContactosAvisados: a.ContactosAvisados}
}

func (s *service) CreateAlert(ctx context.Context, cuentaID uuid.UUID, in pmodels.CreateAlertRequest) (*pmodels.AlertResponse, error) {
	if in.OperationID == uuid.Nil {
		return nil, httpx.Invalid("operationId requerido", map[string]string{"operationId": "requerido"})
	}
	var out *pmodels.AlertResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		existing, err := r.PortalAlertas().GetByOperacionID(ctx, cuentaID, in.OperationID)
		if err == nil {
			out = alertResponse(existing)
			return nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		now := s.now()
		ubic := in.Lat != nil && in.Lon != nil
		a := &models.PortalAlertaAyuda{ID: uuid.New(), CuentaID: cuentaID, OperacionID: in.OperationID, Estado: models.AlertaAyudaActiva, Lat: in.Lat, Lon: in.Lon, UbicacionCompartida: ubic, CreadoEn: now, ActualizadoEn: now}
		if err := r.PortalAlertas().Create(ctx, a); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				ex2, err2 := r.PortalAlertas().GetByOperacionID(ctx, cuentaID, in.OperationID)
				if err2 == nil {
					out = alertResponse(ex2)
					return nil
				}
			}
			return err
		}
		out = alertResponse(a)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) NotifyTrustedContacts(ctx context.Context, cuentaID uuid.UUID, alertID uuid.UUID) (*pmodels.AlertResponse, error) {
	var out *pmodels.AlertResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		a, err := r.PortalAlertas().GetByID(ctx, alertID)
		if err != nil {
			return err
		}
		if a.CuentaID != cuentaID {
			return ErrNotFound
		}
		if a.Estado == models.AlertaAyudaActiva {
			a.ContactosAvisados = true
			a.ActualizadoEn = s.now()
			if err := r.PortalAlertas().Update(ctx, a); err != nil {
				return err
			}
		}
		out = alertResponse(a)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) CancelAlert(ctx context.Context, cuentaID uuid.UUID, alertID uuid.UUID, motivo string) (*pmodels.AlertResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, httpx.Invalid("motivo requerido", map[string]string{"motivo": "requerido"})
	}
	var out *pmodels.AlertResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		a, err := r.PortalAlertas().GetByID(ctx, alertID)
		if err != nil {
			return err
		}
		if a.CuentaID != cuentaID {
			return ErrNotFound
		}
		if a.Estado != models.AlertaAyudaCancelada {
			a.Estado = models.AlertaAyudaCancelada
			a.MotivoCancelacion = &motivo
			a.ActualizadoEn = s.now()
			if err := r.PortalAlertas().Update(ctx, a); err != nil {
				return err
			}
		}
		out = alertResponse(a)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		return r.PortalSesiones().Revoke(ctx, sessionID, s.now())
	})
}
