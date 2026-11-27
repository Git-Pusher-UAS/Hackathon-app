package casefile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

type notesState struct{ notas []models.Nota }

var notesStates = map[*dupRepos]*notesState{}

type fakeNotas struct {
	repository.NotaRepository
	st *notesState
}

func (f fakeNotas) GetByOperacionID(_ context.Context, op uuid.UUID) (*models.Nota, error) {
	for i := range f.st.notas {
		if f.st.notas[i].OperacionID == op {
			return &f.st.notas[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f fakeNotas) EsPrevia(_ context.Context, id, exp, dep uuid.UUID) (bool, error) {
	for _, n := range f.st.notas {
		if n.ID == id && n.ExpedienteID == exp && n.DepartamentoID == dep {
			return true, nil
		}
	}
	return false, nil
}

func (f fakeNotas) TieneCorreccion(_ context.Context, id uuid.UUID) (bool, error) {
	for _, n := range f.st.notas {
		if n.CorrigeNotaID != nil && *n.CorrigeNotaID == id {
			return true, nil
		}
	}
	return false, nil
}

func (f fakeNotas) Create(_ context.Context, n *models.Nota) error {
	f.st.notas = append(f.st.notas, *n)
	return nil
}

type fakeEpisodios struct {
	repository.EpisodioRepository
	id uuid.UUID
}

func (f fakeEpisodios) GetActivo(context.Context, uuid.UUID) (*models.Episodio, error) {
	return &models.Episodio{ID: f.id}, nil
}

func (r *dupRepos) Notas() repository.NotaRepository         { return fakeNotas{st: notesStates[r]} }
func (r *dupRepos) Episodios() repository.EpisodioRepository { return fakeEpisodios{id: uuid.Nil} }

func TestCreateNoteCorreccionUnica(t *testing.T) {
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	exp, dep, actorID := uuid.New(), uuid.New(), uuid.New()
	r := &dupRepos{
		exps:  map[uuid.UUID]*models.Expediente{exp: {ID: exp, Estado: models.ExpedienteAbierto}},
		asigs: []models.Asignacion{{ExpedienteID: exp, PersonalID: actorID, DepartamentoID: dep}},
	}
	assignStates[r] = &assignState{claves: map[uuid.UUID]*models.ClaveRegistro{},
		personal: map[uuid.UUID]*models.Personal{actorID: {ID: actorID, NombreLaboral: "Ana Castro"}}}
	notesStates[r] = &notesState{}
	svc := New(&dupUOW{r}, kr, "secret", "curp")
	actor := repository.ActorContext{ActorID: actorID, RequestID: uuid.New()}
	at := time.Now().Add(-time.Hour)

	orig, err := svc.CreateNote(context.Background(), actor, exp, dep, uuid.New(), casemodels.NoteCreateRequest{Texto: "Primera sesión", OcurridoEn: at})
	if err != nil || orig.Autor != "Ana Castro" {
		t.Fatalf("nota: %+v %v", orig, err)
	}
	id := orig.ID.String()
	if _, err := svc.CreateNote(context.Background(), actor, exp, dep, uuid.New(), casemodels.NoteCreateRequest{Texto: "Corrige la hora", OcurridoEn: at, CorrigeNotaID: &id}); err != nil {
		t.Fatalf("primera corrección: %v", err)
	}
	if _, err := svc.CreateNote(context.Background(), actor, exp, dep, uuid.New(), casemodels.NoteCreateRequest{Texto: "Otra corrección", OcurridoEn: at, CorrigeNotaID: &id}); !errors.Is(err, ErrAlreadyCorrected) {
		t.Fatalf("segunda corrección: %v", err)
	}

	var he *httpx.Error
	largo := strings.Repeat("a", maxNota+1)
	if _, err := svc.CreateNote(context.Background(), actor, exp, dep, uuid.New(), casemodels.NoteCreateRequest{Texto: largo, OcurridoEn: at}); !errors.As(err, &he) || he.Fields["texto"] == "" {
		t.Fatalf("texto largo: %v", err)
	}
	if _, err := svc.CreateNote(context.Background(), actor, exp, dep, uuid.New(), casemodels.NoteCreateRequest{Texto: strings.Repeat("a", maxNota), OcurridoEn: at}); err != nil {
		t.Fatalf("10 000 caracteres deben caber: %v", err)
	}
}
