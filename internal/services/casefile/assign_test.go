package casefile

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

// Extensiones de los fakes de duplicate_test.go para Assign.

type assignState struct {
	sols     map[uuid.UUID]*models.SolicitudVinculacion
	personal map[uuid.UUID]*models.Personal
	claves   map[uuid.UUID]*models.ClaveRegistro
}

var assignStates = map[*dupRepos]*assignState{}

func (f dupExps) GetByID(_ context.Context, id uuid.UUID) (*models.Expediente, error) {
	if e, ok := f.r.exps[id]; ok {
		return e, nil
	}
	return nil, repository.ErrNotFound
}

func (f dupAsigs) Create(_ context.Context, a *models.Asignacion) error {
	f.r.asigs = append(f.r.asigs, *a)
	return nil
}

type assignSols struct {
	repository.SolicitudVinculacionRepository
	st *assignState
}

func (f assignSols) GetByID(_ context.Context, id uuid.UUID) (*models.SolicitudVinculacion, error) {
	if s, ok := f.st.sols[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}

func (f assignSols) Resolver(_ context.Context, id uuid.UUID, estado models.EstadoSolicitudVinculacion, por uuid.UUID, en time.Time, asig *uuid.UUID, v int64) error {
	s := f.st.sols[id]
	if s.Version != v {
		return repository.ErrVersionMismatch
	}
	s.Estado, s.ResueltaPor, s.ResueltaEn, s.AsignacionID = estado, &por, &en, asig
	s.Version++
	return nil
}

type assignPersonal struct {
	repository.PersonalRepository
	st *assignState
}

func (f assignPersonal) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	if p, ok := f.st.personal[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

type assignClaves struct {
	repository.ClaveRegistroRepository
	st *assignState
}

func (f assignClaves) Create(_ context.Context, c *models.ClaveRegistro) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.st.claves[c.ID] = c
	return nil
}

func (f assignClaves) GetByID(_ context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	if c, ok := f.st.claves[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}

func (r *dupRepos) SolicitudesVinculacion() repository.SolicitudVinculacionRepository {
	return assignSols{st: assignStates[r]}
}
func (r *dupRepos) Personal() repository.PersonalRepository {
	return assignPersonal{st: assignStates[r]}
}
func (r *dupRepos) ClavesRegistro() repository.ClaveRegistroRepository {
	return assignClaves{st: assignStates[r]}
}

func TestAssignApprovesRequest(t *testing.T) {
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	exp, dep, psico, solID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	r := &dupRepos{exps: map[uuid.UUID]*models.Expediente{exp: {ID: exp, Estado: models.ExpedienteAbierto}}}
	assignStates[r] = &assignState{
		sols: map[uuid.UUID]*models.SolicitudVinculacion{solID: {ID: solID, ExpedienteID: exp,
			SolicitanteID: psico, Estado: models.SolicitudPendiente, Version: 1}},
		personal: map[uuid.UUID]*models.Personal{psico: {ID: psico, DepartamentoID: dep, Estado: models.PersonalActivo}},
		claves:   map[uuid.UUID]*models.ClaveRegistro{},
	}
	svc := New(&dupUOW{r}, kr, "secret", "curp")
	// Coordinación asigna sin estar asignada al expediente.
	coord := repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	sid := solID.String()
	in := casemodels.AssignRequest{PersonalID: psico.String(), DepartamentoID: dep.String(),
		Motivo: "Solicitud de Psicología aprobada", SolicitudID: &sid}

	res, err := svc.Assign(context.Background(), coord, exp, in)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	sol := assignStates[r].sols[solID]
	if res.PersonalID != psico || res.Fin != nil || sol.Estado != models.SolicitudAprobada ||
		sol.AsignacionID == nil || *sol.AsignacionID != res.ID || *sol.ResueltaPor != coord.ActorID {
		t.Fatalf("asignación %+v, solicitud %+v", res, sol)
	}
	if len(r.audited) != 2 {
		t.Fatalf("auditoría: %v", r.audited)
	}

	// La solicitud ya no está pendiente.
	if _, err := svc.Assign(context.Background(), coord, exp, in); !errors.Is(err, ErrRequestNotPending) {
		t.Fatalf("segunda aprobación: %v", err)
	}
	// Una solicitud de otro expediente no se puede usar.
	otro := uuid.New()
	r.exps[otro] = &models.Expediente{ID: otro, Estado: models.ExpedienteAbierto}
	var ve *ValidationError
	if _, err := svc.Assign(context.Background(), coord, otro, in); !errors.As(err, &ve) {
		t.Fatalf("solicitud de otro expediente: %v", err)
	}
}
