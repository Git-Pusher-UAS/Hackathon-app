package casefile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

type dupUOW struct{ r *dupRepos }

func (f *dupUOW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *dupUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *dupUOW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *dupUOW) Repositories() repository.Repositories { return f.r }

type dupRepos struct {
	repository.Repositories
	indice  map[string]uuid.UUID // hmac -> persona
	exps    map[uuid.UUID]*models.Expediente
	asigs   []models.Asignacion
	deps    map[uuid.UUID]*models.Departamento
	tokens  []models.TokenVinculacion
	audited []string
}

type dupIndice struct {
	repository.IndiceCURPRepository
	r *dupRepos
}

func (f dupIndice) BuscarPersona(_ context.Context, c []repository.IndiceCURPClave) (uuid.UUID, error) {
	if id, ok := f.r.indice[string(c[0].ValorHMAC)]; ok {
		return id, nil
	}
	return uuid.Nil, repository.ErrNotFound
}

type dupExps struct {
	repository.ExpedienteRepository
	r *dupRepos
}

func (f dupExps) GetByPersonaID(_ context.Context, id uuid.UUID) (*models.Expediente, error) {
	for _, e := range f.r.exps {
		if e.PersonaID == id {
			return e, nil
		}
	}
	return nil, repository.ErrNotFound
}

type dupTokens struct {
	repository.TokenVinculacionRepository
	r *dupRepos
}

func (f dupTokens) Create(_ context.Context, t *models.TokenVinculacion) error {
	f.r.tokens = append(f.r.tokens, *t)
	return nil
}

type dupAsigs struct {
	repository.AsignacionRepository
	r *dupRepos
}

func (f dupAsigs) ListByExpediente(_ context.Context, id uuid.UUID, _ bool) ([]models.Asignacion, error) {
	var out []models.Asignacion
	for _, a := range f.r.asigs {
		if a.ExpedienteID == id {
			out = append(out, a)
		}
	}
	return out, nil
}

type dupDeps struct {
	repository.DepartamentoRepository
	r *dupRepos
}

func (f dupDeps) GetByID(_ context.Context, id uuid.UUID) (*models.Departamento, error) {
	return f.r.deps[id], nil
}

type dupAudit struct {
	repository.EventoAuditoriaRepository
	r *dupRepos
}

func (f dupAudit) Append(_ context.Context, e *models.EventoAuditoria) error {
	f.r.audited = append(f.r.audited, e.Accion)
	return nil
}

func (r *dupRepos) IndicesCURP() repository.IndiceCURPRepository             { return dupIndice{r: r} }
func (r *dupRepos) Expedientes() repository.ExpedienteRepository             { return dupExps{r: r} }
func (r *dupRepos) TokensVinculacion() repository.TokenVinculacionRepository { return dupTokens{r: r} }
func (r *dupRepos) Asignaciones() repository.AsignacionRepository            { return dupAsigs{r: r} }
func (r *dupRepos) Departamentos() repository.DepartamentoRepository         { return dupDeps{r: r} }
func (r *dupRepos) EventosAuditoria() repository.EventoAuditoriaRepository   { return dupAudit{r: r} }

func TestDuplicateCheck(t *testing.T) {
	const curp = "GODE561231MDFRRL09"
	svc := New(nil, nil, "secret", "curp-key")
	now := time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	persona, exp, dep := uuid.New(), uuid.New(), uuid.New()
	r := &dupRepos{
		indice: map[string]uuid.UUID{string(svc.curpHMAC(curp)): persona},
		exps:   map[uuid.UUID]*models.Expediente{exp: {ID: exp, PersonaID: persona}},
		asigs:  []models.Asignacion{{ExpedienteID: exp, DepartamentoID: dep}},
		deps:   map[uuid.UUID]*models.Departamento{dep: {ID: dep, Nombre: "Trabajo social"}},
	}
	svc.uow = &dupUOW{r}
	actor := repository.ActorContext{ActorID: uuid.New()}

	res, err := svc.DuplicateCheck(context.Background(), actor, " gode561231mdfrrl09 ")
	if err != nil {
		t.Fatalf("DuplicateCheck: %v", err)
	}
	if !res.Exists || res.Department == nil || *res.Department != "Trabajo social" || res.LinkToken == nil ||
		!strings.HasPrefix(*res.LinkToken, "lt_") || !res.LinkExpiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("respuesta: %+v", res)
	}
	tok := r.tokens[0]
	if tok.ExpedienteID != exp || tok.SolicitanteID != actor.ActorID ||
		string(tok.TokenHash) != string(security.HashOpaqueToken(strings.TrimPrefix(*res.LinkToken, "lt_"))) {
		t.Fatalf("token guardado: %+v", tok)
	}
	if len(r.audited) != 1 {
		t.Fatalf("auditoría: %v", r.audited)
	}

	res, err = svc.DuplicateCheck(context.Background(), actor, "BADD110313HCMLNS09")
	if err != nil || res.Exists || res.LinkToken != nil || len(r.tokens) != 1 {
		t.Fatalf("sin duplicado: %+v %v", res, err)
	}

	var he *httpx.Error
	if _, err := svc.DuplicateCheck(context.Background(), actor, "no-es-curp"); !errors.As(err, &he) || he.Fields["curp"] != "curp_invalid" {
		t.Fatalf("curp inválida: %v", err)
	}
}
