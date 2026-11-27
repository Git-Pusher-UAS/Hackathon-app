package survey

import (
	"context"
	"encoding/json"
	"testing"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

type fakeUOW struct {
	repository.UnitOfWork
	r *fakeRepos
}

func (f *fakeUOW) Repositories() repository.Repositories { return f.r }

type fakeRepos struct {
	repository.Repositories
	enc *fakeEncuestas
}

func (r *fakeRepos) Encuestas() repository.EncuestaRepository { return r.enc }

type fakeEncuestas struct {
	repository.EncuestaRepository
	encuestas []models.Encuesta
	envios    int
	promedio  map[uuid.UUID][2]float64
	conteos   map[uuid.UUID]map[string]int
}

func (f *fakeEncuestas) ListConPreguntas(context.Context) ([]models.Encuesta, error) {
	return f.encuestas, nil
}
func (f *fakeEncuestas) Envios(context.Context, uuid.UUID) (int, error) { return f.envios, nil }
func (f *fakeEncuestas) Promedio(_ context.Context, id uuid.UUID) (float64, int, error) {
	p := f.promedio[id]
	return p[0], int(p[1]), nil
}
func (f *fakeEncuestas) ConteoOpciones(_ context.Context, id uuid.UUID) (map[string]int, error) {
	return f.conteos[id], nil
}

func TestResultsSupresion(t *testing.T) {
	escala, escalaPoca, opcion, abierta := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ops, _ := json.Marshal([]string{"Sí", "No", "No sé"})
	enc := &fakeEncuestas{
		encuestas: []models.Encuesta{{ID: uuid.New(), Titulo: "¿Cómo te atendimos?", Preguntas: []models.PreguntaEncuesta{
			{ID: escala, Clave: "p1", Tipo: models.PreguntaEscala},
			{ID: escalaPoca, Clave: "p2", Tipo: models.PreguntaEscala},
			{ID: opcion, Clave: "p3", Tipo: models.PreguntaOpcion, Opciones: ops},
			{ID: abierta, Clave: "p4", Tipo: models.PreguntaAbierta},
		}}},
		envios:   37,
		promedio: map[uuid.UUID][2]float64{escala: {4.27, 37}, escalaPoca: {5, 3}},
		conteos:  map[uuid.UUID]map[string]int{opcion: {"Sí": 30, "No": 2}},
	}
	res, err := New(&fakeUOW{r: &fakeRepos{enc: enc}}).Results(context.Background())
	if err != nil || len(res) != 1 || res[0].Respuestas == nil || *res[0].Respuestas != 37 {
		t.Fatalf("resultado: %+v %v", res, err)
	}
	p := res[0].Preguntas
	if p[0].Promedio == nil || *p[0].Promedio != 4.3 {
		t.Fatalf("promedio p1: %v", p[0].Promedio)
	}
	if p[1].Promedio != nil {
		t.Fatal("con menos de 5 respuestas el promedio se suprime")
	}
	if *p[2].Conteo[0].Total != 30 || p[2].Conteo[1].Total != nil || p[2].Conteo[2].Total != nil {
		t.Fatalf("conteo: %+v", p[2].Conteo)
	}
	if p[3].Promedio != nil || p[3].Conteo != nil {
		t.Fatal("las abiertas no devuelven contenido")
	}

	enc.envios = 4
	res, _ = New(&fakeUOW{r: &fakeRepos{enc: enc}}).Results(context.Background())
	if res[0].Respuestas != nil {
		t.Fatal("menos de 5 envíos se suprime")
	}
}
