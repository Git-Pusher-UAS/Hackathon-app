package employer

import (
	"context"
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
	emodels "backend-hackathon/internal/models/employer"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrInvalid   = errors.New("invalid")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

type VerifyResult struct {
	Token, CSRFToken string
	EmployerID       uuid.UUID
}

type Service interface {
	Register(ctx context.Context, in emodels.RegistrationRequest) (*emodels.RegistrationResponse, error)
	Login(ctx context.Context, correo string) (*emodels.ChallengeResponse, error)
	Verify(ctx context.Context, challengeID uuid.UUID, code string) (*VerifyResult, error)
	Me(ctx context.Context, id uuid.UUID) (*emodels.EmployerMe, error)
	UpdateRegistration(ctx context.Context, id uuid.UUID, in emodels.UpdateRegistrationRequest) (*emodels.EmployerMe, error)
	ListMyJobs(ctx context.Context, id uuid.UUID) ([]emodels.JobView, error)
	CreateJob(ctx context.Context, id uuid.UUID, in emodels.JobRequest) (*emodels.JobView, error)
	GetMyJob(ctx context.Context, empID, jobID uuid.UUID) (*emodels.JobView, error)
	PatchMyJob(ctx context.Context, empID, jobID uuid.UUID, in emodels.JobRequest) (*emodels.JobView, error)
	ListMyApplications(ctx context.Context, empID, jobID uuid.UUID) ([]emodels.ApplicationView, error)
	GetApplication(ctx context.Context, empID, appID uuid.UUID) (*emodels.ApplicationView, error)
	Interview(ctx context.Context, empID, appID uuid.UUID, in emodels.InterviewRequest) (*emodels.ApplicationView, error)
	Decide(ctx context.Context, empID, appID uuid.UUID, in emodels.DecisionRequest) (*emodels.ApplicationView, error)
}

type service struct {
	uow     repository.UnitOfWork
	now     func() time.Time
	devMode bool
	logf    func(string, ...any)
}

func New(uow repository.UnitOfWork, devMode bool) Service {
	return &service{uow: uow, now: time.Now, devMode: devMode, logf: log.Printf}
}

func randomDigits(n int) (string, error) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b[i] = byte('0' + x.Int64())
	}
	return string(b), nil
}
func hashToken(raw string) []byte { s := sha256.Sum256([]byte(raw)); return s[:] }
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
func maskPhone(p string) string {
	p = strings.TrimSpace(p)
	if len(p) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(p)-4) + p[len(p)-4:]
}

func (s *service) Register(ctx context.Context, in emodels.RegistrationRequest) (*emodels.RegistrationResponse, error) {
	in.Nombre, in.Correo, in.Telefono = strings.TrimSpace(in.Nombre), strings.ToLower(strings.TrimSpace(in.Correo)), strings.TrimSpace(in.Telefono)
	if in.Nombre == "" || in.Correo == "" || in.Telefono == "" {
		return nil, httpx.Invalid("datos inválidos", map[string]string{"nombre": "requerido"})
	}
	var out *emodels.RegistrationResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		if _, err := r.Empleadores().GetByCorreo(ctx, in.Correo); err == nil {
			return ErrConflict
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		e := &models.Empleador{ID: uuid.New(), Nombre: in.Nombre, Correo: in.Correo, Telefono: in.Telefono, Descripcion: strings.TrimSpace(in.Descripcion), Estado: models.EmpleadorPendiente, CreadoEn: s.now(), ActualizadoEn: s.now()}
		if err := r.Empleadores().Create(ctx, e); err != nil {
			return err
		}
		out = &emodels.RegistrationResponse{ID: e.ID, Estado: e.Estado}
		return nil
	})
	return out, err
}

func (s *service) createChallenge(ctx context.Context, r repository.Repositories, e *models.Empleador) (*emodels.ChallengeResponse, error) {
	code, err := randomDigits(6)
	if err != nil {
		return nil, err
	}
	d := &models.EmpleadorDesafioOTP{ID: uuid.New(), EmpleadorID: e.ID, CodigoHash: hashToken(code), VenceEn: s.now().Add(10 * time.Minute), CreadoEn: s.now()}
	if err := r.EmpleadorDesafios().Create(ctx, d); err != nil {
		return nil, err
	}
	if s.devMode {
		s.logf("[DEV OTP] employer challenge=%s correo=%s telefono=%s code=%s", d.ID, e.Correo, maskPhone(e.Telefono), code)
	}
	return &emodels.ChallengeResponse{ChallengeID: d.ID, TelefonoMascara: maskPhone(e.Telefono)}, nil
}

func (s *service) Login(ctx context.Context, correo string) (*emodels.ChallengeResponse, error) {
	correo = strings.ToLower(strings.TrimSpace(correo))
	if correo == "" {
		return nil, httpx.Invalid("correo requerido", map[string]string{"correo": "requerido"})
	}
	var out *emodels.ChallengeResponse
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Empleadores().GetByCorreo(ctx, correo)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		out, err = s.createChallenge(ctx, r, e)
		return err
	})
	return out, err
}

func (s *service) Verify(ctx context.Context, challengeID uuid.UUID, code string) (*VerifyResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, httpx.Invalid("code requerido", map[string]string{"code": "requerido"})
	}
	var out *VerifyResult
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		d, err := r.EmpleadorDesafios().GetByID(ctx, challengeID)
		if err != nil {
			return ErrNotFound
		}
		now := s.now()
		if d.ConsumidoEn != nil || !d.VenceEn.After(now) || d.Intentos >= 5 {
			return ErrInvalid
		}
		sum := sha256.Sum256([]byte(code))
		if subtle.ConstantTimeCompare(sum[:], d.CodigoHash) != 1 {
			d.Intentos++
			_ = r.EmpleadorDesafios().Save(ctx, d)
			return ErrInvalid
		}
		d.ConsumidoEn = &now
		if err := r.EmpleadorDesafios().Save(ctx, d); err != nil {
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
		ses := &models.EmpleadorSesion{ID: uuid.New(), EmpleadorID: d.EmpleadorID, TokenHash: hashToken(token), CSRFHash: hashToken(csrf), UltimaActividad: now, VenceAbsoluto: now.Add(24 * time.Hour), CreadoEn: now}
		if err := r.EmpleadorSesiones().Create(ctx, ses); err != nil {
			return err
		}
		out = &VerifyResult{Token: ses.ID.String() + "." + token, CSRFToken: csrf, EmployerID: d.EmpleadorID}
		return nil
	})
	return out, err
}

func (s *service) Me(ctx context.Context, id uuid.UUID) (*emodels.EmployerMe, error) {
	e, err := s.uow.Repositories().Empleadores().GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &emodels.EmployerMe{ID: e.ID, Nombre: e.Nombre, Correo: e.Correo, Telefono: e.Telefono, Descripcion: e.Descripcion, Estado: e.Estado, MotivoRevision: e.MotivoRevision}, nil
}

func (s *service) UpdateRegistration(ctx context.Context, id uuid.UUID, in emodels.UpdateRegistrationRequest) (*emodels.EmployerMe, error) {
	var out *emodels.EmployerMe
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Empleadores().GetByID(ctx, id)
		if err != nil {
			return err
		}
		if strings.TrimSpace(in.Nombre) != "" {
			e.Nombre = strings.TrimSpace(in.Nombre)
		}
		if strings.TrimSpace(in.Telefono) != "" {
			e.Telefono = strings.TrimSpace(in.Telefono)
		}
		e.Descripcion = strings.TrimSpace(in.Descripcion)
		e.ActualizadoEn = s.now()
		if err := r.Empleadores().Save(ctx, e); err != nil {
			return err
		}
		out = &emodels.EmployerMe{ID: e.ID, Nombre: e.Nombre, Correo: e.Correo, Telefono: e.Telefono, Descripcion: e.Descripcion, Estado: e.Estado, MotivoRevision: e.MotivoRevision}
		return nil
	})
	return out, err
}

func jobView(v *models.Vacante, c int) emodels.JobView {
	return emodels.JobView{ID: v.ID, Puesto: v.Puesto, Empresa: v.Empresa, Sector: v.Sector, Horario: v.Horario, TipoEmpleo: v.TipoEmpleo, Ubicacion: v.Ubicacion, Descripcion: v.Descripcion, Requisitos: v.Requisitos, Estado: v.Estado, PublicadaEn: v.PublicadaEn, Actualizada: v.ActualizadaEn, Postulaciones: c}
}

func (s *service) ListMyJobs(ctx context.Context, id uuid.UUID) ([]emodels.JobView, error) {
	r := s.uow.Repositories()
	page, err := r.Vacantes().List(ctx, repository.VacanteFilter{EmpleadorID: &id}, repository.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(page.Items))
	for _, v := range page.Items {
		ids = append(ids, v.ID)
	}
	counts, _ := r.Vacantes().CountPostulaciones(ctx, ids)
	out := make([]emodels.JobView, 0, len(page.Items))
	for i := range page.Items {
		out = append(out, jobView(&page.Items[i], counts[page.Items[i].ID]))
	}
	return out, nil
}

func (s *service) CreateJob(ctx context.Context, id uuid.UUID, in emodels.JobRequest) (*emodels.JobView, error) {
	var out *emodels.JobView
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Empleadores().GetByID(ctx, id)
		if err != nil {
			return err
		}
		now := s.now()
		v := &models.Vacante{ID: uuid.New(), EmpleadorID: &id, Puesto: strings.TrimSpace(in.Puesto), Empresa: e.Nombre, Sector: in.Sector, Horario: in.Horario, TipoEmpleo: in.TipoEmpleo, Ubicacion: in.Ubicacion, Descripcion: in.Descripcion, Requisitos: in.Requisitos, Estado: models.VacantePendiente, PublicadaEn: now, ActualizadaEn: now, CreadoEn: now}
		if err := r.Vacantes().Create(ctx, v); err != nil {
			return err
		}
		j := jobView(v, 0)
		out = &j
		return nil
	})
	return out, err
}

func (s *service) GetMyJob(ctx context.Context, empID, jobID uuid.UUID) (*emodels.JobView, error) {
	r := s.uow.Repositories()
	v, err := r.Vacantes().GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if v.EmpleadorID == nil || *v.EmpleadorID != empID {
		return nil, ErrNotFound
	}
	counts, _ := r.Vacantes().CountPostulaciones(ctx, []uuid.UUID{jobID})
	j := jobView(v, counts[jobID])
	return &j, nil
}

func (s *service) PatchMyJob(ctx context.Context, empID, jobID uuid.UUID, in emodels.JobRequest) (*emodels.JobView, error) {
	var out *emodels.JobView
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		v, err := r.Vacantes().GetByID(ctx, jobID)
		if err != nil {
			return err
		}
		if v.EmpleadorID == nil || *v.EmpleadorID != empID {
			return ErrNotFound
		}
		if strings.TrimSpace(in.Puesto) != "" {
			v.Puesto = strings.TrimSpace(in.Puesto)
		}
		if in.Sector != "" {
			v.Sector = in.Sector
		}
		if in.Horario != "" {
			v.Horario = in.Horario
		}
		if in.TipoEmpleo != "" {
			v.TipoEmpleo = in.TipoEmpleo
		}
		if in.Ubicacion != "" {
			v.Ubicacion = in.Ubicacion
		}
		if in.Descripcion != "" {
			v.Descripcion = in.Descripcion
		}
		if in.Requisitos != "" {
			v.Requisitos = in.Requisitos
		}
		v.ActualizadaEn = s.now()
		if err := r.Vacantes().Save(ctx, v); err != nil {
			return err
		}
		counts, _ := r.Vacantes().CountPostulaciones(ctx, []uuid.UUID{jobID})
		j := jobView(v, counts[jobID])
		out = &j
		return nil
	})
	return out, err
}

func (s *service) ListMyApplications(ctx context.Context, empID, jobID uuid.UUID) ([]emodels.ApplicationView, error) {
	r := s.uow.Repositories()
	v, err := r.Vacantes().GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if v.EmpleadorID == nil || *v.EmpleadorID != empID {
		return nil, ErrNotFound
	}
	rows, err := r.Postulaciones().ListByVacante(ctx, jobID)
	if err != nil {
		return nil, err
	}
	out := make([]emodels.ApplicationView, 0, len(rows))
	for _, p := range rows {
		out = append(out, emodels.ApplicationView{ID: p.ID, VacanteID: p.VacanteID, Estado: p.Estado, CreadoEn: p.CreadoEn, ContactoLiberado: p.ContactoLiberado})
	}
	return out, nil
}

func (s *service) GetApplication(ctx context.Context, empID, appID uuid.UUID) (*emodels.ApplicationView, error) {
	r := s.uow.Repositories()
	p, err := r.Postulaciones().GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	v, err := r.Vacantes().GetByID(ctx, p.VacanteID)
	if err != nil {
		return nil, err
	}
	if v.EmpleadorID == nil || *v.EmpleadorID != empID {
		return nil, ErrNotFound
	}
	view := &emodels.ApplicationView{ID: p.ID, VacanteID: p.VacanteID, Estado: p.Estado, CreadoEn: p.CreadoEn, ContactoLiberado: p.ContactoLiberado}
	if p.ContactoLiberado {
		if cuenta, err := r.PortalCuentas().GetByID(ctx, p.UsuariaID); err == nil {
			name := cuenta.Nombre
			view.Nombre = &name
			contact := cuenta.Telefono
			view.Contacto = &contact
		}
	}
	return view, nil
}

func (s *service) Interview(ctx context.Context, empID, appID uuid.UUID, in emodels.InterviewRequest) (*emodels.ApplicationView, error) {
	var out *emodels.ApplicationView
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Postulaciones().GetByID(ctx, appID)
		if err != nil {
			return err
		}
		v, err := r.Vacantes().GetByID(ctx, p.VacanteID)
		if err != nil {
			return err
		}
		if v.EmpleadorID == nil || *v.EmpleadorID != empID {
			return ErrNotFound
		}
		if in.Fecha != "" {
			if t, err := time.Parse(time.RFC3339, in.Fecha); err == nil {
				p.EntrevistaEn = &t
			}
		}
		if in.Lugar != "" {
			p.EntrevistaLugar = &in.Lugar
		}
		if in.Notas != "" {
			p.EntrevistaNotas = &in.Notas
		}
		p.Estado = models.PostulacionEntrevistaInvitada
		p.ActualizadoEn = s.now()
		if err := r.Postulaciones().Save(ctx, p); err != nil {
			return err
		}
		out = &emodels.ApplicationView{ID: p.ID, VacanteID: p.VacanteID, Estado: p.Estado, CreadoEn: p.CreadoEn, ContactoLiberado: p.ContactoLiberado}
		return nil
	})
	return out, err
}

func (s *service) Decide(ctx context.Context, empID, appID uuid.UUID, in emodels.DecisionRequest) (*emodels.ApplicationView, error) {
	var out *emodels.ApplicationView
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Postulaciones().GetByID(ctx, appID)
		if err != nil {
			return err
		}
		v, err := r.Vacantes().GetByID(ctx, p.VacanteID)
		if err != nil {
			return err
		}
		if v.EmpleadorID == nil || *v.EmpleadorID != empID {
			return ErrNotFound
		}
		switch strings.ToLower(strings.TrimSpace(in.Decision)) {
		case "accept", "accepted":
			p.Estado = models.PostulacionAceptada
		case "reject", "rejected":
			p.Estado = models.PostulacionRechazada
		case "hire", "hired":
			p.Estado = models.PostulacionContratada
		default:
			return httpx.Invalid("decision invalida", map[string]string{"decision": "valor_invalido"})
		}
		if strings.TrimSpace(in.Motivo) != "" {
			m := strings.TrimSpace(in.Motivo)
			p.DecisionMotivo = &m
		}
		p.ActualizadoEn = s.now()
		if err := r.Postulaciones().Save(ctx, p); err != nil {
			return err
		}
		out = &emodels.ApplicationView{ID: p.ID, VacanteID: p.VacanteID, Estado: p.Estado, CreadoEn: p.CreadoEn, ContactoLiberado: p.ContactoLiberado}
		return nil
	})
	return out, err
}
