package idempotency

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type fakeUOW struct{ r *fakeRepos }

func (f *fakeUOW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.r)
}
func (f *fakeUOW) Repositories() repository.Repositories { return f.r }

type fakeRepos struct {
	repository.Repositories
	ops    *fakeOps
	claves *fakeClaves
}

func (r *fakeRepos) Operaciones() repository.OperacionRepository        { return r.ops }
func (r *fakeRepos) ClavesRegistro() repository.ClaveRegistroRepository { return r.claves }

type fakeOps struct {
	repository.OperacionRepository
	items []models.Operacion
}

func (f *fakeOps) Find(_ context.Context, actor uuid.UUID, tipo string, op uuid.UUID) (*models.Operacion, error) {
	for i := range f.items {
		o := f.items[i]
		if o.ActorID == actor && o.TipoOperacion == tipo && o.OperationID == op {
			return &o, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeOps) Create(_ context.Context, o *models.Operacion) error {
	f.items = append(f.items, *o)
	return nil
}

type fakeClaves struct {
	repository.ClaveRegistroRepository
	m map[uuid.UUID]*models.ClaveRegistro
}

func (f *fakeClaves) Create(_ context.Context, c *models.ClaveRegistro) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	cp := *c
	f.m[c.ID] = &cp
	return nil
}

func (f *fakeClaves) GetByID(_ context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	if c, ok := f.m[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}

func newApp(t *testing.T, required bool) (*fiber.App, *int, *fakeOps) {
	t.Helper()
	kr, err := security.NewKeyring(make([]byte, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	ops := &fakeOps{}
	uow := &fakeUOW{&fakeRepos{ops: ops, claves: &fakeClaves{m: map[uuid.UUID]*models.ClaveRegistro{}}}}
	calls := 0
	actor := repository.ActorContext{ActorID: uuid.New(), RequestID: uuid.New()}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.Write})
	app.Post("/things",
		func(c *fiber.Ctx) error { c.Locals("actor", actor); return c.Next() },
		New(Config{UOW: uow, Keyring: kr, Secret: "s"}, required),
		func(c *fiber.Ctx) error {
			calls++
			c.Set(fiber.HeaderETag, `"v1"`)
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{"n": calls, "key": c.Get(HeaderAlias)})
		})
	return app, &calls, ops
}

func post(t *testing.T, app *fiber.App, body string, headers map[string]string) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/things", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, int(5*time.Second/time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), resp.Header.Get(fiber.HeaderETag)
}

func TestReplayAndMismatch(t *testing.T) {
	app, calls, ops := newApp(t, true)
	key := uuid.NewString()
	h := map[string]string{HeaderKey: key}

	st, body, etag := post(t, app, `{"a":1}`, h)
	if st != 201 || *calls != 1 || etag != `"v1"` || !strings.Contains(body, key) {
		t.Fatalf("primera: %d %s %q calls=%d", st, body, etag, *calls)
	}
	if len(ops.items) != 1 || ops.items[0].HTTPStatus != 201 {
		t.Fatalf("operación guardada: %+v", ops.items)
	}

	// Misma clave y mismo cuerpo: misma respuesta sin ejecutar el handler.
	st2, body2, etag2 := post(t, app, `{"a":1}`, h)
	if st2 != 201 || body2 != body || etag2 != `"v1"` || *calls != 1 {
		t.Fatalf("repetición: %d %s calls=%d", st2, body2, *calls)
	}

	// Misma clave con otro cuerpo: 409.
	st3, body3, _ := post(t, app, `{"a":2}`, h)
	if st3 != 409 || !strings.Contains(body3, "idempotency_mismatch") || *calls != 1 {
		t.Fatalf("otro cuerpo: %d %s", st3, body3)
	}

	// El alias X-Operation-Id es la misma clave.
	st4, _, _ := post(t, app, `{"a":1}`, map[string]string{HeaderAlias: key})
	if st4 != 201 || *calls != 1 {
		t.Fatalf("alias: %d calls=%d", st4, *calls)
	}
}

func TestKeyValidation(t *testing.T) {
	app, calls, _ := newApp(t, true)
	cases := map[string]map[string]string{
		"sin clave":        {},
		"clave no uuid":    {HeaderKey: "abc"},
		"claves distintas": {HeaderKey: uuid.NewString(), HeaderAlias: uuid.NewString()},
	}
	for name, h := range cases {
		st, body, _ := post(t, app, `{}`, h)
		if st != 422 || !strings.Contains(body, `"fields"`) {
			t.Errorf("%s: %d %s", name, st, body)
		}
	}
	if *calls != 0 {
		t.Fatalf("el handler no debía correr: %d", *calls)
	}

	opt, optCalls, ops := newApp(t, false)
	if st, _, _ := post(t, opt, `{}`, nil); st != 201 || *optCalls != 1 || len(ops.items) != 0 {
		t.Fatalf("clave opcional ausente: %d calls=%d ops=%d", st, *optCalls, len(ops.items))
	}
}
