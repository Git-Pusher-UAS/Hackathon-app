package adminjobs

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"github.com/google/uuid"
)

type Service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) *Service { return &Service{uow: uow, now: time.Now} }

func (s *Service) ListJobs(ctx context.Context) ([]models.Vacante, error) {
	p, err := s.uow.Repositories().Vacantes().List(ctx, repository.VacanteFilter{}, repository.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}
	return p.Items, nil
}
func (s *Service) CreateJob(ctx context.Context, in map[string]any) (*models.Vacante, error) {
	now := s.now()
	str := func(k string) string { v, _ := in[k].(string); return strings.TrimSpace(v) }
	v := &models.Vacante{ID: uuid.New(), Puesto: str("puesto"), Empresa: str("empresa"), Sector: str("sector"), Horario: str("horario"), TipoEmpleo: str("tipoEmpleo"), Ubicacion: str("ubicacion"), Descripcion: str("descripcion"), Requisitos: str("requisitos"), Estado: models.VacantePendiente, PublicadaEn: now, ActualizadaEn: now, CreadoEn: now}
	if v.Empresa == "" {
		v.Empresa = "IMM"
	}
	if err := s.uow.Repositories().Vacantes().Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (*models.Vacante, error) {
	return s.uow.Repositories().Vacantes().GetByID(ctx, id)
}
func (s *Service) ReviewJob(ctx context.Context, id uuid.UUID, decision, motivo string) (*models.Vacante, error) {
	r := s.uow.Repositories()
	v, err := r.Vacantes().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approve", "approved", "publish", "published":
		v.Estado = models.VacantePublicada
	case "reject", "rejected":
		v.Estado = models.VacanteRechazada
	default:
		v.Estado = models.VacantePendiente
	}
	if strings.TrimSpace(motivo) != "" {
		m := strings.TrimSpace(motivo)
		v.MotivoRevision = &m
	}
	v.ActualizadaEn = s.now()
	if err := r.Vacantes().Save(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) ListEmployers(ctx context.Context) ([]models.Empleador, error) {
	return s.uow.Repositories().Empleadores().List(ctx, nil)
}
func (s *Service) ReviewEmployer(ctx context.Context, id uuid.UUID, decision, motivo string) (*models.Empleador, error) {
	r := s.uow.Repositories()
	e, err := r.Empleadores().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approve", "approved":
		e.Estado = models.EmpleadorAprobado
	case "reject", "rejected":
		e.Estado = models.EmpleadorRechazado
	default:
		e.Estado = models.EmpleadorPendiente
	}
	if strings.TrimSpace(motivo) != "" {
		m := strings.TrimSpace(motivo)
		e.MotivoRevision = &m
	}
	e.ActualizadoEn = s.now()
	if err := r.Empleadores().Save(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}
func (s *Service) ListHires(ctx context.Context) ([]models.Postulacion, error) {
	p, err := s.uow.Repositories().Vacantes().List(ctx, repository.VacanteFilter{}, repository.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}
	var out []models.Postulacion
	for _, v := range p.Items {
		ps, _ := s.uow.Repositories().Postulaciones().ListByVacante(ctx, v.ID)
		for _, pp := range ps {
			if pp.Estado == models.PostulacionContratada {
				out = append(out, pp)
			}
		}
	}
	return out, nil
}
func (s *Service) ContactRelease(ctx context.Context, actor repository.ActorContext, appID uuid.UUID, motivo string, campos []string) (*models.Postulacion, error) {
	var out *models.Postulacion
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Postulaciones().GetByID(ctx, appID)
		if err != nil {
			return err
		}
		now := s.now()
		p.ContactoLiberado = true
		p.ContactoLiberadoEn = &now
		p.ContactoLiberadoPor = &actor.ActorID
		p.ActualizadoEn = now
		if strings.TrimSpace(motivo) != "" {
			m := strings.TrimSpace(motivo)
			p.DecisionMotivo = &m
		}
		if err := r.Postulaciones().Save(ctx, p); err != nil {
			return err
		}
		meta, _ := json.Marshal(map[string]any{"actorId": actor.ActorID, "timestamp": now.UTC(), "applicationId": p.ID, "jobId": p.VacanteID, "candidateAccountId": p.UsuariaID, "motivo": strings.TrimSpace(motivo), "camposLiberados": campos})
		if err := audit.RecordWithMeta(ctx, r, &actor.ActorID, "admin.contact_release", "postulacion", models.ResultadoExito, &p.ID, actor.RequestID, actor.IPOrigen, meta); err != nil {
			return err
		}
		out = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
