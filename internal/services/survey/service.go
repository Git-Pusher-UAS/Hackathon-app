package survey

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	smodels "backend-hackathon/internal/models/survey"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const K = 5

var (
	ErrNotFound = errors.New("survey not found")
	ErrConflict = errors.New("already submitted")
)

type Service interface {
	Pending(ctx context.Context, usuariaID uuid.UUID) ([]smodels.Encuesta, error)
	Get(ctx context.Context, id uuid.UUID) (*smodels.Encuesta, error)
	Submit(ctx context.Context, usuariaID, encuestaID uuid.UUID, in smodels.SubmitRequest) error
	Results(ctx context.Context) ([]smodels.Resultado, error)
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

func toDTO(e *models.Encuesta) smodels.Encuesta {
	out := smodels.Encuesta{ID: e.ID, Titulo: e.Titulo, Minutos: e.Minutos, Preguntas: make([]smodels.Pregunta, 0, len(e.Preguntas))}
	for _, p := range e.Preguntas {
		var opts []string
		if len(p.Opciones) > 0 {
			_ = json.Unmarshal(p.Opciones, &opts)
		}
		out.Preguntas = append(out.Preguntas, smodels.Pregunta{ID: p.Clave, Texto: p.Texto, Tipo: p.Tipo, Opciones: opts})
	}
	return out
}

func (s *service) Pending(ctx context.Context, usuariaID uuid.UUID) ([]smodels.Encuesta, error) {
	var out []smodels.Encuesta
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		rows, err := r.Encuestas().ListPendingByUsuaria(ctx, usuariaID)
		if err != nil {
			return err
		}
		out = make([]smodels.Encuesta, 0, len(rows))
		for i := range rows {
			out = append(out, toDTO(&rows[i]))
		}
		return nil
	})
	return out, err
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*smodels.Encuesta, error) {
	var out *smodels.Encuesta
	err := s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Encuestas().GetByIDWithPreguntas(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		dto := toDTO(e)
		out = &dto
		return nil
	})
	return out, err
}

func (s *service) Submit(ctx context.Context, usuariaID, encuestaID uuid.UUID, in smodels.SubmitRequest) error {
	if in.OperationID == uuid.Nil {
		return httpx.Invalid("operationId requerido", map[string]string{"operationId": "requerido"})
	}
	if in.Respuestas == nil {
		return httpx.Invalid("respuestas requeridas", map[string]string{"respuestas": "requerido"})
	}
	return s.uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		e, err := r.Encuestas().GetByIDWithPreguntas(ctx, encuestaID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		asig, err := r.Encuestas().GetAsignacion(ctx, encuestaID, usuariaID)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if err == nil && asig.Contestada {
			if asig.OperationID != nil && *asig.OperationID == in.OperationID {
				return nil
			}
			return ErrConflict
		}
		if byOp, err := r.Encuestas().GetAsignacionByOperationID(ctx, in.OperationID); err == nil {
			if byOp.EncuestaID == encuestaID && byOp.UsuariaID == usuariaID && byOp.Contestada {
				return nil
			}
			return ErrConflict
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}

		now := s.now()
		envioID := uuid.New()
		rows := make([]models.RespuestaEncuesta, 0, len(e.Preguntas))
		for _, p := range e.Preguntas {
			val, ok := in.Respuestas[p.Clave]
			if !ok {
				continue
			}
			row := models.RespuestaEncuesta{ID: uuid.New(), EncuestaID: encuestaID, PreguntaID: p.ID, EnvioID: envioID, CreadoEn: now}
			switch p.Tipo {
			case models.PreguntaEscala:
				f, ok := toFloat(val)
				if !ok {
					return httpx.Invalid("respuesta invalida", map[string]string{p.Clave: "numero_requerido"})
				}
				row.Numero = &f
			case models.PreguntaOpcion, models.PreguntaAbierta:
				sv, ok := val.(string)
				if !ok {
					return httpx.Invalid("respuesta invalida", map[string]string{p.Clave: "texto_requerido"})
				}
				row.Texto = &sv
			}
			rows = append(rows, row)
		}
		if err := r.Encuestas().CreateRespuestas(ctx, rows); err != nil {
			return err
		}
		if asig == nil {
			asig = &models.AsignacionEncuesta{ID: uuid.New(), EncuestaID: encuestaID, UsuariaID: usuariaID, CreadoEn: now}
		}
		asig.Contestada = true
		asig.OperationID = &in.OperationID
		asig.EnvioID = &envioID
		asig.ContestadaEn = &now
		return r.Encuestas().SaveAsignacion(ctx, asig)
	})
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func suprimir(n int) *int {
	if n < K {
		return nil
	}
	v := n
	return &v
}

func (s *service) Results(ctx context.Context) ([]smodels.Resultado, error) {
	r := s.uow.Repositories().Encuestas()
	encuestas, err := r.ListConPreguntas(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]smodels.Resultado, 0, len(encuestas))
	for _, e := range encuestas {
		envios, err := r.Envios(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		res := smodels.Resultado{EncuestaID: e.ID.String(), Titulo: e.Titulo, Respuestas: suprimir(envios), Preguntas: make([]smodels.ResultadoPregunta, 0, len(e.Preguntas))}
		for _, p := range e.Preguntas {
			item := smodels.ResultadoPregunta{ID: p.Clave, Texto: p.Texto, Tipo: p.Tipo}
			switch p.Tipo {
			case models.PreguntaEscala:
				prom, n, err := r.Promedio(ctx, p.ID)
				if err != nil {
					return nil, err
				}
				if n >= K {
					v := math.Round(prom*10) / 10
					item.Promedio = &v
				}
			case models.PreguntaOpcion:
				conteos, err := r.ConteoOpciones(ctx, p.ID)
				if err != nil {
					return nil, err
				}
				var opciones []string
				_ = json.Unmarshal(p.Opciones, &opciones)
				item.Conteo = make([]smodels.Conteo, 0, len(opciones))
				for _, o := range opciones {
					item.Conteo = append(item.Conteo, smodels.Conteo{Opcion: o, Total: suprimir(conteos[o])})
				}
			}
			res.Preguntas = append(res.Preguntas, item)
		}
		out = append(out, res)
	}
	return out, nil
}
