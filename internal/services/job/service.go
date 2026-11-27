package job

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	jmodels "backend-hackathon/internal/models/job"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("job not found")
	ErrDuplicate = errors.New("duplicate application")
)

type Service interface {
	List(ctx context.Context, f repository.VacanteFilter, portalUsuariaID *uuid.UUID) ([]jmodels.Job, error)
	Get(ctx context.Context, id uuid.UUID, portalUsuariaID *uuid.UUID) (*jmodels.Job, error)
	Create(ctx context.Context, in jmodels.CreateRequest) (*jmodels.Job, error)
	Apply(ctx context.Context, vacanteID, usuariaID uuid.UUID) error
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

func view(v *models.Vacante, postulaciones int, postulada *bool) jmodels.Job {
	p := postulaciones
	return jmodels.Job{ID: v.ID, Puesto: v.Puesto, Empresa: v.Empresa, Sector: v.Sector, Horario: v.Horario, TipoEmpleo: v.TipoEmpleo, Ubicacion: v.Ubicacion, Descripcion: v.Descripcion, Requisitos: v.Requisitos, PublicadaEn: v.PublicadaEn.UTC(), Actualizada: v.ActualizadaEn.UTC(), Postulada: postulada, Postulaciones: &p}
}

func (s *service) List(ctx context.Context, f repository.VacanteFilter, portalUsuariaID *uuid.UUID) ([]jmodels.Job, error) {
	r := s.uow.Repositories()
	rp, err := r.Vacantes().List(ctx, f, repository.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rp.Items))
	for _, v := range rp.Items {
		ids = append(ids, v.ID)
	}
	counts, err := r.Vacantes().CountPostulaciones(ctx, ids)
	if err != nil {
		return nil, err
	}
	var applied map[uuid.UUID]bool
	if portalUsuariaID != nil {
		applied, err = r.Postulaciones().ExistsMany(ctx, ids, *portalUsuariaID)
		if err != nil {
			return nil, err
		}
	}
	out := make([]jmodels.Job, 0, len(rp.Items))
	for i := range rp.Items {
		var post *bool
		if portalUsuariaID != nil {
			v := applied[rp.Items[i].ID]
			post = &v
		}
		out = append(out, view(&rp.Items[i], counts[rp.Items[i].ID], post))
	}
	return out, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID, portalUsuariaID *uuid.UUID) (*jmodels.Job, error) {
	r := s.uow.Repositories()
	v, err := r.Vacantes().GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	counts, err := r.Vacantes().CountPostulaciones(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	var post *bool
	if portalUsuariaID != nil {
		ok, err := r.Postulaciones().Exists(ctx, id, *portalUsuariaID)
		if err != nil {
			return nil, err
		}
		post = &ok
	}
	j := view(v, counts[id], post)
	return &j, nil
}

func (s *service) Create(ctx context.Context, in jmodels.CreateRequest) (*jmodels.Job, error) {
	fields := map[string]string{}
	reqStr := func(name, val string) {
		if strings.TrimSpace(val) == "" {
			fields[name] = "requerido"
		}
	}
	reqStr("puesto", in.Puesto)
	reqStr("empresa", in.Empresa)
	reqStr("sector", in.Sector)
	reqStr("horario", in.Horario)
	reqStr("tipoEmpleo", in.TipoEmpleo)
	reqStr("ubicacion", in.Ubicacion)
	reqStr("descripcion", in.Descripcion)
	reqStr("requisitos", in.Requisitos)
	if len(fields) > 0 {
		return nil, httpx.Invalid("vacante invalida", fields)
	}
	now := s.now().UTC()
	v := &models.Vacante{ID: uuid.New(), Puesto: strings.TrimSpace(in.Puesto), Empresa: strings.TrimSpace(in.Empresa), Sector: strings.TrimSpace(in.Sector), Horario: strings.TrimSpace(in.Horario), TipoEmpleo: strings.TrimSpace(in.TipoEmpleo), Ubicacion: strings.TrimSpace(in.Ubicacion), Descripcion: strings.TrimSpace(in.Descripcion), Requisitos: strings.TrimSpace(in.Requisitos), PublicadaEn: now, ActualizadaEn: now, CreadoEn: now}
	if err := s.uow.Repositories().Vacantes().Create(ctx, v); err != nil {
		return nil, err
	}
	j := view(v, 0, nil)
	return &j, nil
}

func (s *service) Apply(ctx context.Context, vacanteID, usuariaID uuid.UUID) error {
	return s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		if _, err := r.Vacantes().GetByID(ctx, vacanteID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		exists, err := r.Postulaciones().Exists(ctx, vacanteID, usuariaID)
		if err != nil {
			return err
		}
		if exists {
			return ErrDuplicate
		}
		err = r.Postulaciones().Create(ctx, &models.Postulacion{ID: uuid.New(), VacanteID: vacanteID, UsuariaID: usuariaID, CreadoEn: s.now()})
		if errors.Is(err, repository.ErrConflict) {
			return ErrDuplicate
		}
		return err
	})
}
