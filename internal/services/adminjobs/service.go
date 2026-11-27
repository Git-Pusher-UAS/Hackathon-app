package adminjobs

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/metrics"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"github.com/google/uuid"
)

type Service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) *Service { return &Service{uow: uow, now: time.Now} }

type CreateJobRequest struct{ Puesto, Empresa, Sector, Horario, TipoEmpleo, Ubicacion, Descripcion, Requisitos string }

func mapRepoErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*httpx.Error); ok {
		return err
	}
	switch err {
	case repository.ErrNotFound:
		return httpx.E(404, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case repository.ErrConflict:
		return httpx.E(409, "CONFLICT", "conflicto")
	default:
		return err
	}
}

func (s *Service) ListJobs(ctx context.Context, actor repository.ActorContext) ([]models.Vacante, error) {
	var out []models.Vacante
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Vacantes().List(ctx, repository.VacanteFilter{}, repository.PageRequest{Limit: 100})
		if err != nil {
			return err
		}
		out = p.Items
		return nil
	})
	return out, mapRepoErr(err)
}

func (s *Service) CreateJob(ctx context.Context, actor repository.ActorContext, in CreateJobRequest) (*models.Vacante, error) {
	trim := strings.TrimSpace
	fields := map[string]string{}
	req := func(name, v string) string {
		v = trim(v)
		if v == "" {
			fields[name] = "requerido"
		}
		return v
	}
	puesto := req("puesto", in.Puesto)
	empresa := trim(in.Empresa)
	sector := req("sector", in.Sector)
	horario := req("horario", in.Horario)
	tipo := req("tipoEmpleo", in.TipoEmpleo)
	ubic := req("ubicacion", in.Ubicacion)
	desc := req("descripcion", in.Descripcion)
	reqs := req("requisitos", in.Requisitos)
	if empresa == "" {
		empresa = "IMM"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("vacante invalida", fields)
	}
	now := s.now()
	v := &models.Vacante{ID: uuid.New(), Puesto: puesto, Empresa: empresa, Sector: sector, Horario: horario, TipoEmpleo: tipo, Ubicacion: ubic, Descripcion: desc, Requisitos: reqs, Estado: models.VacantePendiente, PublicadaEn: now, ActualizadaEn: now, CreadoEn: now}
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error { return r.Vacantes().Create(ctx, v) })
	return v, mapRepoErr(err)
}

func (s *Service) GetJob(ctx context.Context, actor repository.ActorContext, id uuid.UUID) (*models.Vacante, error) {
	var v *models.Vacante
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		var e error
		v, e = r.Vacantes().GetByID(ctx, id)
		return e
	})
	return v, mapRepoErr(err)
}

func (s *Service) ReviewJob(ctx context.Context, actor repository.ActorContext, id uuid.UUID, decision, motivo string) (*models.Vacante, error) {
	var out *models.Vacante
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		v, err := r.Vacantes().GetByID(ctx, id)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(decision)) {
		case "approve", "approved", "publish", "published":
			v.Estado = models.VacantePublicada
		case "reject", "rejected":
			v.Estado = models.VacanteRechazada
		default:
			return httpx.Invalid("decision invalida", map[string]string{"decision": "valor_invalido"})
		}
		if strings.TrimSpace(motivo) != "" {
			m := strings.TrimSpace(motivo)
			v.MotivoRevision = &m
		}
		v.ActualizadaEn = s.now()
		if err := r.Vacantes().Save(ctx, v); err != nil {
			return err
		}
		out = v
		return nil
	})
	return out, mapRepoErr(err)
}

func (s *Service) ListEmployers(ctx context.Context, actor repository.ActorContext) ([]models.Empleador, error) {
	var out []models.Empleador
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		var e error
		out, e = r.Empleadores().List(ctx, nil)
		return e
	})
	return out, mapRepoErr(err)
}

func (s *Service) ReviewEmployer(ctx context.Context, actor repository.ActorContext, id uuid.UUID, decision, motivo string) (*models.Empleador, error) {
	var out *models.Empleador
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Empleadores().GetByID(ctx, id)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(decision)) {
		case "approve", "approved":
			e.Estado = models.EmpleadorAprobado
		case "reject", "rejected":
			e.Estado = models.EmpleadorRechazado
		default:
			return httpx.Invalid("decision invalida", map[string]string{"decision": "valor_invalido"})
		}
		if strings.TrimSpace(motivo) != "" {
			m := strings.TrimSpace(motivo)
			e.MotivoRevision = &m
		}
		e.ActualizadoEn = s.now()
		if err := r.Empleadores().Save(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	return out, mapRepoErr(err)
}

func (s *Service) ListHires(ctx context.Context, actor repository.ActorContext) ([]models.Postulacion, error) {
	var out []models.Postulacion
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Vacantes().List(ctx, repository.VacanteFilter{}, repository.PageRequest{Limit: 100})
		if err != nil {
			return err
		}
		for _, v := range p.Items {
			ps, _ := r.Postulaciones().ListByVacante(ctx, v.ID)
			for _, pp := range ps {
				if pp.Estado == models.PostulacionContratada {
					out = append(out, pp)
				}
			}
		}
		return nil
	})
	return out, mapRepoErr(err)
}

func (s *Service) ContactRelease(ctx context.Context, actor repository.ActorContext, appID uuid.UUID, motivo string, campos []string) (*models.Postulacion, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, httpx.Invalid("motivo requerido", map[string]string{"motivo": "requerido"})
	}
	if len(campos) == 0 {
		campos = []string{"nombre", "telefono"}
	}
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
		p.DecisionMotivo = &motivo
		if err := r.Postulaciones().Save(ctx, p); err != nil {
			return err
		}
		meta, _ := json.Marshal(map[string]any{"actorId": actor.ActorID, "timestamp": now.UTC(), "applicationId": p.ID, "jobId": p.VacanteID, "candidateAccountId": p.UsuariaID, "motivo": motivo, "camposLiberados": campos})
		if err := audit.RecordWithMeta(ctx, r, &actor.ActorID, "admin.contact_release", "postulacion", models.ResultadoExito, &p.ID, actor.RequestID, actor.IPOrigen, meta); err != nil {
			metrics.AuditAppendErrors.Add(1)
			return err
		}
		metrics.ContactReleaseTotal.Add(1)
		out = p
		return nil
	})
	return out, mapRepoErr(err)
}
