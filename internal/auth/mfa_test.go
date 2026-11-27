package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/url"
	"testing"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func TestPasoValido(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "test",
		AccountName: "admin",
		SecretSize:  32,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}

	now := time.Unix(1_800_000_000, 0)
	base := now.Unix() / 30

	casos := []struct {
		nombre string
		at     time.Time
		delta  int64
	}{
		{"ventana actual", now, 0},
		{"ventana anterior", now.Add(-30 * time.Second), -1},
		{"ventana siguiente", now.Add(30 * time.Second), 1},
	}
	for _, c := range casos {
		code, err := totp.GenerateCode(key.Secret(), c.at)
		if err != nil {
			t.Fatalf("%s: GenerateCode: %v", c.nombre, err)
		}
		paso, ok := pasoValido(code, key.Secret(), now)
		if !ok {
			t.Fatalf("%s: código válido rechazado", c.nombre)
		}
		if paso != base+c.delta {
			t.Fatalf("%s: paso = %d, esperado %d", c.nombre, paso, base+c.delta)
		}
	}

	if _, ok := pasoValido("12345", key.Secret(), now); ok {
		t.Fatal("un código con longitud incorrecta fue aceptado")
	}
	if _, ok := pasoValido("1234567", key.Secret(), now); ok {
		t.Fatal("un código con longitud incorrecta fue aceptado")
	}
}

func TestCSRFValido(t *testing.T) {
	token, err := randomToken(tamanoTokenCSRF)
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	hash := sha256.Sum256([]byte(token))

	if !CSRFValido(hash[:], token) {
		t.Fatal("el token correcto no validó")
	}
	if CSRFValido(hash[:], token+"x") {
		t.Fatal("un token alterado validó")
	}
	if CSRFValido(nil, token) {
		t.Fatal("un hash vacío validó")
	}
	if CSRFValido(hash[:], "") {
		t.Fatal("un token vacío validó")
	}
}

func TestCifrarDescifrar(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	s := &Service{mfaKey: key}

	cifrado, err := s.cifrar([]byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("cifrar: %v", err)
	}
	if string(cifrado) == "JBSWY3DPEHPK3PXP" {
		t.Fatal("el secreto se guardó en claro")
	}

	plano, err := s.descifrar(cifrado)
	if err != nil {
		t.Fatalf("descifrar: %v", err)
	}
	if string(plano) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("plano = %q", plano)
	}

	cifrado[len(cifrado)-1] ^= 0xff
	if _, err := s.descifrar(cifrado); err == nil {
		t.Fatal("un texto cifrado alterado descifró")
	}
}

// ---------- flujo enroll -> verify -> login con challenge ----------

func TestFlujoMFACompleto(t *testing.T) {
	ctx := context.Background()
	svc, repos, _ := nuevoServicioDePrueba(t)
	login := func() *Challenge {
		t.Helper()
		ch, err := svc.Login(ctx, LoginInput{Usuario: "admin", Password: "admin123", IP: "127.0.0.1", UserAgent: "test", RequestID: uuid.New()})
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		return ch
	}

	// 1. Sin segundo factor: el login nunca da sesión, pide enrolar.
	ch := login()
	if ch.Next != NextEnroll || ch.ChallengeID == "" {
		t.Fatalf("primer login: %+v, quiero next=enroll", ch)
	}
	if _, err := svc.EnrollMFA(ctx, "otro-reto", uuid.New(), "127.0.0.1"); !errors.Is(err, ErrDesafioInvalido) {
		t.Fatalf("enrolar con reto desconocido: %v", err)
	}
	enroll, err := svc.EnrollMFA(ctx, ch.ChallengeID, uuid.New(), "127.0.0.1")
	if err != nil {
		t.Fatalf("EnrollMFA: %v", err)
	}
	u, err := url.Parse(enroll.OtpauthURI)
	if err != nil || u.Query().Get("secret") == "" || enroll.EnrollmentID == uuid.Nil {
		t.Fatalf("enrolamiento incompleto: %+v", enroll)
	}
	secret := u.Query().Get("secret")

	now := time.Now()
	code, _ := totp.GenerateCode(secret, now)
	// Sin enrollmentId no se confirma.
	if _, err := svc.VerifyMFA(ctx, VerifyMFAInput{Challenge: ch.ChallengeID, Code: code, IP: "127.0.0.1", RequestID: uuid.New()}); !errors.Is(err, ErrMFANoIniciado) {
		t.Fatalf("verificar sin enrollmentId: %v", err)
	}
	ses, err := svc.VerifyMFA(ctx, VerifyMFAInput{Challenge: ch.ChallengeID, EnrollmentID: &enroll.EnrollmentID, Code: code, IP: "127.0.0.1", UserAgent: "test", RequestID: uuid.New()})
	if err != nil {
		t.Fatalf("VerifyMFA (enrolamiento): %v", err)
	}
	if ses.Token == "" || ses.CSRF == "" || ses.Usuario.Usuario != "admin" || !ses.IdleExpiresAt.After(now) {
		t.Fatalf("sesión incompleta: %+v", ses)
	}

	// 2. Con segundo factor activo: next=totp.
	ch2 := login()
	if ch2.Next != NextTOTP {
		t.Fatalf("segundo login: next=%s", ch2.Next)
	}
	if _, err := svc.EnrollMFA(ctx, ch2.ChallengeID, uuid.New(), "127.0.0.1"); !errors.Is(err, ErrDesafioInvalido) {
		t.Fatalf("un reto totp no sirve para enrolar: %v", err)
	}
	// El paso usado en la activación ya no puede reutilizarse: usamos el siguiente.
	code2, _ := totp.GenerateCode(secret, now.Add(30*time.Second))
	res, err := svc.VerifyMFA(ctx, VerifyMFAInput{Challenge: ch2.ChallengeID, Code: code2, IP: "127.0.0.1", UserAgent: "test", RequestID: uuid.New()})
	if err != nil {
		t.Fatalf("VerifyMFA (login): %v", err)
	}

	// El reto es de un solo uso.
	if _, err := svc.VerifyMFA(ctx, VerifyMFAInput{Challenge: ch2.ChallengeID, Code: code2, IP: "127.0.0.1", RequestID: uuid.New()}); !errors.Is(err, ErrDesafioInvalido) {
		t.Fatalf("reutilizar reto: esperaba ErrDesafioInvalido, got %v", err)
	}

	// CSRF rota sobre la sesión emitida.
	claims, err := ParseToken("secreto-de-prueba", res.Token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	csrf, err := svc.RotarCSRF(ctx, claims.SessionID)
	if err != nil {
		t.Fatalf("RotarCSRF: %v", err)
	}
	sesion := repos.sesiones.porID[claims.SessionID]
	if sesion == nil || !CSRFValido(sesion.CSRFHash, csrf.CSRF) {
		t.Fatal("el hash CSRF no se actualizó en la sesión")
	}
}

func nuevoServicioDePrueba(t *testing.T) (*Service, *fakeRepos, *models.Personal) {
	t.Helper()

	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	p := &models.Personal{
		ID:              uuid.New(),
		Usuario:         "admin",
		NombreLaboral:   "Admin",
		RolID:           uuid.New(),
		PasswordHash:    hash,
		Estado:          models.PersonalActivo,
		VersionPermisos: 1,
	}

	repos := &fakeRepos{
		personal: &fakePersonalRepo{
			porID:      map[uuid.UUID]*models.Personal{p.ID: p},
			porUsuario: map[string]*models.Personal{"admin": p},
		},
		mfa:       &fakeMFARepo{porPersonal: map[uuid.UUID]*models.MFAPersonal{}},
		sesiones:  &fakeSesionRepo{porID: map[uuid.UUID]*models.SesionPersonal{}},
		contextos: &fakeContextoRepo{porHash: map[string]*models.ContextoAuth{}},
		intentos:  &fakeIntentosRepo{},
		roles:     &fakeRolRepo{},
		permisos:  &fakePermisoRepo{},
		auditoria: &fakeAuditoriaRepo{},
	}
	uow := &fakeUOW{repos: repos}

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	svc, err := NewService(uow, "secreto-de-prueba", time.Hour, key)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, repos, p
}

type fakeUOW struct{ repos repository.Repositories }

func (f *fakeUOW) Repositories() repository.Repositories { return f.repos }

func (f *fakeUOW) Do(_ context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(context.Background(), f.repos)
}

func (f *fakeUOW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.repos)
}

func (f *fakeUOW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, f.repos)
}

type fakeRepos struct {
	repository.Repositories
	personal  *fakePersonalRepo
	mfa       *fakeMFARepo
	sesiones  *fakeSesionRepo
	contextos *fakeContextoRepo
	intentos  *fakeIntentosRepo
	roles     *fakeRolRepo
	permisos  *fakePermisoRepo
	auditoria *fakeAuditoriaRepo
}

func (f *fakeRepos) Personal() repository.PersonalRepository               { return f.personal }
func (f *fakeRepos) MFA() repository.MFAPersonalRepository                 { return f.mfa }
func (f *fakeRepos) Sesiones() repository.SesionPersonalRepository         { return f.sesiones }
func (f *fakeRepos) ContextosAuth() repository.ContextoAuthRepository      { return f.contextos }
func (f *fakeRepos) ControlIntentos() repository.ControlIntentosRepository { return f.intentos }
func (f *fakeRepos) Roles() repository.RolRepository                       { return f.roles }
func (f *fakeRepos) Permisos() repository.PermisoRepository                { return f.permisos }
func (f *fakeRepos) EventosAuditoria() repository.EventoAuditoriaRepository {
	return f.auditoria
}

type fakePersonalRepo struct {
	repository.PersonalRepository
	porID      map[uuid.UUID]*models.Personal
	porUsuario map[string]*models.Personal
}

func (r *fakePersonalRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Personal, error) {
	if p, ok := r.porID[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakePersonalRepo) GetByUsuario(_ context.Context, usuario string) (*models.Personal, error) {
	if p, ok := r.porUsuario[usuario]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakePersonalRepo) ReiniciarFallos(_ context.Context, _ uuid.UUID) error { return nil }

func (r *fakePersonalRepo) RegistrarFallo(_ context.Context, _ uuid.UUID, _ *time.Time) (int, *time.Time, error) {
	return 1, nil, nil
}

type fakeMFARepo struct {
	repository.MFAPersonalRepository
	porPersonal map[uuid.UUID]*models.MFAPersonal
}

func (r *fakeMFARepo) GetByPersonalID(_ context.Context, personalID uuid.UUID) (*models.MFAPersonal, error) {
	if m, ok := r.porPersonal[personalID]; ok {
		return m, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakeMFARepo) DeletePendiente(_ context.Context, personalID uuid.UUID) error {
	m, ok := r.porPersonal[personalID]
	if !ok || m.Estado == models.MFAActivo {
		return repository.ErrNotFound
	}
	delete(r.porPersonal, personalID)
	return nil
}

func (r *fakeMFARepo) Create(_ context.Context, m *models.MFAPersonal) error {
	if _, ok := r.porPersonal[m.PersonalID]; ok {
		return repository.ErrConflict
	}
	r.porPersonal[m.PersonalID] = m
	return nil
}

func (r *fakeMFARepo) Activar(_ context.Context, id uuid.UUID, confirmadoEn time.Time, paso int64) error {
	for _, m := range r.porPersonal {
		if m.ID == id {
			m.Estado = models.MFAActivo
			m.ConfirmadoEn = &confirmadoEn
			m.UltimoPasoUsado = &paso
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *fakeMFARepo) ConsumirPaso(_ context.Context, id uuid.UUID, paso int64) (bool, error) {
	for _, m := range r.porPersonal {
		if m.ID != id || m.Estado != models.MFAActivo {
			continue
		}
		if m.UltimoPasoUsado != nil && *m.UltimoPasoUsado >= paso {
			return false, nil
		}
		p := paso
		m.UltimoPasoUsado = &p
		return true, nil
	}
	return false, nil
}

type fakeSesionRepo struct {
	repository.SesionPersonalRepository
	porID map[uuid.UUID]*models.SesionPersonal
}

func (r *fakeSesionRepo) Create(_ context.Context, s *models.SesionPersonal) error {
	r.porID[s.ID] = s
	return nil
}

func (r *fakeSesionRepo) ActualizarCSRF(_ context.Context, id uuid.UUID, csrfHash []byte) error {
	s, ok := r.porID[id]
	if !ok {
		return repository.ErrNotFound
	}
	s.CSRFHash = csrfHash
	return nil
}

type fakeContextoRepo struct {
	repository.ContextoAuthRepository
	porHash map[string]*models.ContextoAuth
}

func (r *fakeContextoRepo) Create(_ context.Context, c *models.ContextoAuth) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	r.porHash[string(c.TokenHash)] = c
	return nil
}

func (r *fakeContextoRepo) GetByTokenHash(_ context.Context, tokenHash []byte) (*models.ContextoAuth, error) {
	if c, ok := r.porHash[string(tokenHash)]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakeContextoRepo) IncrementarIntentos(_ context.Context, id uuid.UUID) (int, error) {
	for _, c := range r.porHash {
		if c.ID == id {
			c.Intentos++
			return c.Intentos, nil
		}
	}
	return 0, repository.ErrNotFound
}

func (r *fakeContextoRepo) Consumir(_ context.Context, id uuid.UUID, at time.Time) (bool, error) {
	for _, c := range r.porHash {
		if c.ID == id && c.ConsumidoEn == nil {
			c.ConsumidoEn = &at
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeContextoRepo) VincularEnrolamiento(_ context.Context, id, mfaID uuid.UUID) error {
	for _, c := range r.porHash {
		if c.ID == id {
			c.EnrolamientoID = &mfaID
			return nil
		}
	}
	return repository.ErrNotFound
}

type fakeIntentosRepo struct {
	repository.ControlIntentosRepository
}

func (r *fakeIntentosRepo) EstaBloqueado(_ context.Context, _ string, _ []byte, _ time.Time) (bool, *time.Time, error) {
	return false, nil, nil
}

func (r *fakeIntentosRepo) Incrementar(_ context.Context, ambito string, claveHMAC []byte, ventanaInicio time.Time) (*models.ControlIntentos, error) {
	return &models.ControlIntentos{Ambito: ambito, ClaveHMAC: claveHMAC, VentanaInicio: ventanaInicio, Contador: 1}, nil
}

func (r *fakeIntentosRepo) Bloquear(_ context.Context, _ string, _ []byte, _ time.Time, _ time.Time) error {
	return nil
}

type fakeRolRepo struct {
	repository.RolRepository
}

func (r *fakeRolRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Rol, error) {
	return &models.Rol{ID: id, Codigo: "admin"}, nil
}

type fakePermisoRepo struct {
	repository.PermisoRepository
}

func (r *fakePermisoRepo) CodigosEfectivos(_ context.Context, _ uuid.UUID, _ time.Time) ([]string, error) {
	return []string{"admin"}, nil
}

type fakeAuditoriaRepo struct {
	repository.EventoAuditoriaRepository
}

func (r *fakeAuditoriaRepo) Append(_ context.Context, _ *models.EventoAuditoria) error { return nil }
