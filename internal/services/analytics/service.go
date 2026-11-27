// Package analytics implementa los agregados del tablero (§13.2). Cuenta
// expedientes abiertos en el periodo y suprime toda cifra menor a 5.
package analytics

import (
	"context"
	"fmt"
	"time"

	"backend-hackathon/internal/httpx"
	anmodels "backend-hackathon/internal/models/analytics"
	repository "backend-hackathon/internal/repository/repositories"
)

const metodologia = "Expedientes abiertos en el periodo. Cifras menores a 5 se suprimen."

type Service interface {
	Summary(ctx context.Context, period string) (*anmodels.Summary, error)
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

// zona es la hora local de Los Mochis (UTC-7, sin horario de verano).
var zona = func() *time.Location {
	if loc, err := time.LoadLocation("America/Mazatlan"); err == nil {
		return loc
	}
	return time.FixedZone("UTC-7", -7*60*60)
}()

// Corte es la medianoche local de hoy: los agregados cubren hasta ahí.
func Corte(now time.Time) time.Time {
	l := now.In(zona)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, zona).UTC()
}

// Rango devuelve el periodo actual y el anterior de la misma duración.
func Rango(period string, corte time.Time) (actual, anterior repository.TimeRange, err error) {
	var desde time.Time
	switch period {
	case "", "30d":
		desde = corte.AddDate(0, 0, -30)
	case "6m":
		desde = corte.AddDate(0, -6, 0)
	case "12m":
		desde = corte.AddDate(0, -12, 0)
	default:
		return actual, anterior, httpx.Invalid("period inválido", map[string]string{"period": "valor_invalido"})
	}
	actual = repository.TimeRange{Desde: desde, Hasta: corte}
	anterior = repository.TimeRange{Desde: desde.Add(-corte.Sub(desde)), Hasta: desde}
	return actual, anterior, nil
}

// Suprimir devuelve nil si la cifra es menor a k.
func Suprimir(n int) *int {
	if n < anmodels.SupresionK {
		return nil
	}
	return &n
}

func serie(cs []repository.Conteo) []anmodels.Serie {
	out := make([]anmodels.Serie, 0, len(cs))
	for _, c := range cs {
		out = append(out, anmodels.Serie{Etiqueta: c.Etiqueta, Valor: Suprimir(c.Valor)})
	}
	return out
}

func indicador(clave, titulo string, actual, anterior int) anmodels.Indicador {
	ind := anmodels.Indicador{Clave: clave, Titulo: titulo, Valor: Suprimir(actual)}
	switch {
	case ind.Valor == nil:
		ind.Suprimido, ind.Detalle = true, "Suprimido"
	case Suprimir(anterior) == nil:
		// No se publica la diferencia: revelaría la cifra suprimida.
		ind.Detalle = "Sin comparación con el periodo anterior"
	default:
		ind.Detalle = fmt.Sprintf("%+d vs. periodo anterior", actual-anterior)
	}
	return ind
}

func (s *service) Summary(ctx context.Context, period string) (*anmodels.Summary, error) {
	corte := Corte(s.now())
	actual, anterior, err := Rango(period, corte)
	if err != nil {
		return nil, err
	}
	r := s.uow.Repositories().Indicadores()
	abiertos, err := r.Expedientes(ctx, actual)
	if err != nil {
		return nil, err
	}
	abiertosAnt, err := r.Expedientes(ctx, anterior)
	if err != nil {
		return nil, err
	}
	canal, err := r.Canalizaciones(ctx, actual)
	if err != nil {
		return nil, err
	}
	canalAnt, err := r.Canalizaciones(ctx, anterior)
	if err != nil {
		return nil, err
	}
	mes, err := r.PorMes(ctx, actual)
	if err != nil {
		return nil, err
	}
	origen, err := r.PorOrigen(ctx, actual)
	if err != nil {
		return nil, err
	}
	area, err := r.PorArea(ctx, actual)
	if err != nil {
		return nil, err
	}
	return &anmodels.Summary{
		Corte:       corte,
		Metodologia: metodologia,
		Indicadores: []anmodels.Indicador{
			indicador("abiertos", "Expedientes abiertos", abiertos, abiertosAnt),
			indicador("canalizaciones", "Canalizaciones", canal, canalAnt),
		},
		PorMes:    serie(mes),
		PorOrigen: serie(origen),
		PorArea:   serie(area),
	}, nil
}
