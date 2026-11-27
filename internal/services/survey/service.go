// Package survey implementa los resultados agregados de encuestas (§12.4).
package survey

import (
	"context"
	"encoding/json"
	"math"

	"backend-hackathon/internal/models"
	smodels "backend-hackathon/internal/models/survey"
	repository "backend-hackathon/internal/repository/repositories"
)

// K es el mínimo de respuestas para publicar una cifra.
const K = 5

type Service interface {
	Results(ctx context.Context) ([]smodels.Resultado, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

func suprimir(n int) *int {
	if n < K {
		return nil
	}
	return &n
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
		res := smodels.Resultado{EncuestaID: e.ID.String(), Titulo: e.Titulo, Respuestas: suprimir(envios),
			Preguntas: make([]smodels.Pregunta, 0, len(e.Preguntas))}
		for _, p := range e.Preguntas {
			item := smodels.Pregunta{ID: p.Clave, Texto: p.Texto, Tipo: p.Tipo}
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
