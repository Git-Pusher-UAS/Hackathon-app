// Package user implementa las reglas de negocio de /api/v1/users.
package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	usermodels "backend-hackathon/internal/models/user"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const (
	adminPermission = "admin"

	cursorScopeUsers    = "users.list"
	cursorScopeSessions = "users.me.sessions"

	justificationLifetime = 365 * 24 * time.Hour
)

// Service expone los casos de uso de las rutas /api/v1/users.
type Service interface {
	Me(ctx context.Context, actorID, sessionID uuid.UUID) (*usermodels.MeResponse, error)
	ListMySessions(ctx context.Context, actorID, currentSessionID uuid.UUID, page repository.PageRequest) (*usermodels.Page[usermodels.SessionSummary], error)
	RevokeMySession(ctx context.Context, actor repository.ActorContext, actorID, sessionID uuid.UUID) error
	List(ctx context.Context, actorID uuid.UUID, q usermodels.ListQuery, page repository.PageRequest) (*usermodels.Page[usermodels.StaffSummary], error)
	Create(ctx context.Context, actor repository.ActorContext, in usermodels.CreateRequest) (*usermodels.CreateResponse, error)
	Update(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, in usermodels.UpdateRequest, ifMatch string) (*usermodels.StaffSummary, error)
	RevokeAllSessions(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string) error
	ResetMFA(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string, verificacionPresencial bool) (uuid.UUID, error)
}

type service struct {
	uow     repository.UnitOfWork
	keyring *security.Keyring
	secret  string
	now     func() time.Time
}

// New construye el servicio de usuarios.
func New(uow repository.UnitOfWork, keyring *security.Keyring, secret string) *service {
	return &service{uow: uow, keyring: keyring, secret: secret, now: time.Now}
}

func (s *service) Me(ctx context.Context, actorID, sessionID uuid.UUID) (*usermodels.MeResponse, error) {
	repos := s.uow.Repositories()

	p, err := repos.Personal().GetByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthenticated
		}
		return nil, err
	}
	now := s.now()
	if p.Estado != models.PersonalActivo || (p.BloqueadoHasta != nil && p.BloqueadoHasta.After(now)) {
		return nil, ErrUnauthenticated
	}

	sesion, err := repos.Sesiones().GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthenticated
		}
		return nil, err
	}
	if sesion.TitularID != actorID || sesion.RevocadoEn != nil || !sesion.VenceAbsoluto.After(now) || !sesion.UltimaActividad.Add(auth.IdleTimeout).After(now) {
		return nil, ErrUnauthenticated
	}

	rol, err := repos.Roles().GetByID(ctx, p.RolID)
	if err != nil {
		return nil, fmt.Errorf("usuario: buscar rol: %w", err)
	}
	permisos, err := repos.Permisos().CodigosEfectivos(ctx, p.ID, now)
	if err != nil {
		return nil, fmt.Errorf("usuario: permisos efectivos: %w", err)
	}
	if permisos == nil {
		permisos = []string{}
	}

	return &usermodels.MeResponse{
		Actor: usermodels.Actor{
			ID:           p.ID,
			Rol:          rol.Codigo,
			DepartmentID: p.DepartamentoID,
			DisplayName:  p.NombreLaboral,
			Permissions:  permisos,
		},
		ExpiresAt:     sesion.VenceAbsoluto.UTC(),
		IdleExpiresAt: sesion.UltimaActividad.Add(auth.IdleTimeout).UTC(),
	}, nil
}

func (s *service) ListMySessions(ctx context.Context, actorID, currentSessionID uuid.UUID, page repository.PageRequest) (*usermodels.Page[usermodels.SessionSummary], error) {
	filterHash := httpx.FilterHash()
	plainCursor, err := httpx.DecodeCursor(s.secret, page.Cursor, actorID, cursorScopeSessions, filterHash)
	if err != nil {
		return nil, &ValidationError{Msg: "cursor inválido"}
	}

	repoPage, err := s.uow.Repositories().Sesiones().ListByTitular(ctx, actorID, true, s.now(), repository.PageRequest{
		Limit:  page.Limit,
		Cursor: plainCursor,
	})
	if err != nil {
		return nil, err
	}

	items := make([]usermodels.SessionSummary, 0, len(repoPage.Items))
	for _, sesion := range repoPage.Items {
		items = append(items, usermodels.SessionSummary{
			ID:             sesion.ID,
			CreatedAt:      sesion.CreadoEn.UTC(),
			LastActivityAt: sesion.UltimaActividad.UTC(),
			ExpiresAt:      sesion.VenceAbsoluto.UTC(),
			DeviceLabel:    neutralDeviceLabel(sesion.Dispositivo),
			Current:        sesion.ID == currentSessionID,
		})
	}

	var next *string
	if repoPage.NextCursor != "" {
		cursor := httpx.EncodeCursor(s.secret, actorID, cursorScopeSessions, filterHash, repoPage.NextCursor)
		next = &cursor
	}
	return &usermodels.Page[usermodels.SessionSummary]{Items: items, NextCursor: next}, nil
}

func (s *service) RevokeMySession(ctx context.Context, actor repository.ActorContext, actorID, sessionID uuid.UUID) error {
	err := s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		sesion, err := repos.Sesiones().GetByID(ctx, sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		if sesion.TitularID != actorID {
			return ErrNotFound
		}
		if sesion.RevocadoEn == nil {
			if err := repos.Sesiones().Revocar(ctx, sessionID, s.now()); err != nil && !errors.Is(err, repository.ErrNotFound) {
				return err
			}
		}
		return audit.RecordResource(ctx, repos, &actorID, "users.session.revoke", "sesion", models.ResultadoExito, &sessionID, actor.RequestID, actor.IPOrigen)
	})
	if errors.Is(err, ErrNotFound) {
		s.auditDenied(ctx, actor, "users.session.revoke", "sesion", &sessionID)
	}
	return err
}

func (s *service) List(ctx context.Context, actorID uuid.UUID, q usermodels.ListQuery, page repository.PageRequest) (*usermodels.Page[usermodels.StaffSummary], error) {
	q.Q, q.Role, q.Status = strings.TrimSpace(q.Q), strings.TrimSpace(q.Role), strings.TrimSpace(q.Status)
	filterHash := httpx.FilterHash(q.Q, q.Role, q.Status, strconv.FormatBool(q.SoloAtencion))
	plainCursor, err := httpx.DecodeCursor(s.secret, page.Cursor, actorID, cursorScopeUsers, filterHash)
	if err != nil {
		return nil, &ValidationError{Msg: "cursor inválido"}
	}

	repos := s.uow.Repositories()
	filter := repository.PersonalFilter{Busqueda: q.Q}
	switch q.Status {
	case "":
	case string(models.PersonalActivo), string(models.PersonalSuspendido):
		estado := models.EstadoPersonal(q.Status)
		filter.Estado = &estado
	default:
		return nil, &ValidationError{Msg: "status debe ser activo o suspendido"}
	}
	if q.Role != "" {
		rol, err := repos.Roles().GetByCodigo(ctx, q.Role)
		if errors.Is(err, repository.ErrNotFound) {
			return &usermodels.Page[usermodels.StaffSummary]{Items: []usermodels.StaffSummary{}}, nil
		}
		if err != nil {
			return nil, err
		}
		filter.RolID = &rol.ID
	}
	if q.SoloAtencion {
		activo := models.PersonalActivo
		filter.Estado = &activo
		deps, err := repos.Departamentos().List(ctx, true)
		if err != nil {
			return nil, err
		}
		for _, d := range deps {
			if d.Asistencial {
				filter.DepartamentoIDs = append(filter.DepartamentoIDs, d.ID)
			}
		}
		if len(filter.DepartamentoIDs) == 0 {
			return &usermodels.Page[usermodels.StaffSummary]{Items: []usermodels.StaffSummary{}}, nil
		}
	}
	repoPage, err := repos.Personal().List(ctx, filter, repository.PageRequest{
		Limit:  page.Limit,
		Cursor: plainCursor,
	})
	if err != nil {
		return nil, err
	}

	roles, err := repos.Roles().List(ctx, false)
	if err != nil {
		return nil, err
	}
	roleCodes := make(map[uuid.UUID]string, len(roles))
	for _, rol := range roles {
		roleCodes[rol.ID] = rol.Codigo
	}

	ids := make([]uuid.UUID, 0, len(repoPage.Items))
	for _, p := range repoPage.Items {
		ids = append(ids, p.ID)
	}
	ultimos, err := repos.Sesiones().UltimoAcceso(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]usermodels.StaffSummary, 0, len(repoPage.Items))
	for i := range repoPage.Items {
		p := &repoPage.Items[i]
		item := usermodels.StaffSummary{
			ID:             p.ID,
			Usuario:        p.Usuario,
			NombreLaboral:  p.NombreLaboral,
			Rol:            roleCodes[p.RolID],
			DepartamentoID: p.DepartamentoID,
			Estado:         p.Estado,
			VersionTag:     httpx.EncodeVersionTag(s.secret, p.Version),
		}
		if t, ok := ultimos[p.ID]; ok {
			item.UltimoAcceso = &t
		}
		items = append(items, item)
	}

	var next *string
	if repoPage.NextCursor != "" {
		cursor := httpx.EncodeCursor(s.secret, actorID, cursorScopeUsers, filterHash, repoPage.NextCursor)
		next = &cursor
	}
	return &usermodels.Page[usermodels.StaffSummary]{Items: items, NextCursor: next}, nil
}

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in usermodels.CreateRequest) (*usermodels.CreateResponse, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}

	var created *models.Personal
	var rolCode string
	var activacion usermodels.Activacion
	err := s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		rol, err := repos.Roles().GetByCodigo(ctx, strings.TrimSpace(in.Rol))
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return &ValidationError{Msg: "rol inexistente"}
			}
			return err
		}
		if !rol.Activo {
			return &ValidationError{Msg: "rol inactivo"}
		}

		departamentoID, _ := uuid.Parse(strings.TrimSpace(in.DepartamentoID))
		departamento, err := repos.Departamentos().GetByID(ctx, departamentoID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return &ValidationError{Msg: "departamento inexistente"}
			}
			return err
		}
		if !departamento.Activo {
			return &ValidationError{Msg: "departamento inactivo"}
		}

		usuario := strings.ToLower(strings.TrimSpace(in.Usuario))
		if _, err := repos.Personal().GetByUsuario(ctx, usuario); err == nil {
			return ErrDuplicateUsername
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}

		// Contraseña aleatoria que nadie conoce hasta la activación.
		inicial, err := auth.HashPassword(uuid.NewString() + uuid.NewString())
		if err != nil {
			return err
		}

		actorModel := &models.Actor{Tipo: models.ActorPersonal, Activo: true}
		if err := repos.Actores().Create(ctx, actorModel); err != nil {
			return err
		}

		personal := &models.Personal{
			ID:                     actorModel.ID,
			Usuario:                usuario,
			NombreLaboral:          strings.TrimSpace(in.NombreLaboral),
			RolID:                  rol.ID,
			DepartamentoID:         departamento.ID,
			PasswordHash:           inicial,
			RequiereCambioPassword: true,
			Estado:                 models.PersonalActivo,
			VersionPermisos:        1,
			Version:                1,
		}
		if err := repos.Personal().Create(ctx, personal); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrDuplicateUsername
			}
			return err
		}

		codigo, vence, err := auth.NuevaActivacion(ctx, repos, s.secret, personal.ID, s.now())
		if err != nil {
			return err
		}
		activacion = usermodels.Activacion{Codigo: codigo, ExpiresAt: vence}
		created = personal
		rolCode = rol.Codigo
		return audit.Record(ctx, repos, &actor.ActorID, "users.create", models.ResultadoExito, &personal.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &usermodels.CreateResponse{Cuenta: *s.summary(created, rolCode), Activacion: activacion}, nil
}

func (s *service) Update(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, in usermodels.UpdateRequest, ifMatch string) (*usermodels.StaffSummary, error) {
	if err := validateUpdate(in); err != nil {
		return nil, err
	}
	if ifMatch == "" {
		return nil, ErrPreconditionRequired
	}
	expectedVersion, err := httpx.DecodeVersionTag(s.secret, ifMatch)
	if err != nil {
		return nil, ErrVersionMismatch
	}
	if targetID == actor.ActorID {
		s.auditDenied(ctx, actor, "users.update", "personal", &targetID)
		return nil, ErrSelfElevation
	}

	var updated *models.Personal
	var rolCode string
	err = s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		p, err := repos.Personal().GetByID(ctx, targetID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		if p.Version != expectedVersion {
			return ErrVersionMismatch
		}

		newRoleID := p.RolID
		newRoleCode := ""
		currentRole, err := repos.Roles().GetByID(ctx, p.RolID)
		if err != nil {
			return fmt.Errorf("usuario: rol actual: %w", err)
		}
		newRoleCode = currentRole.Codigo
		if in.Rol != nil {
			rol, err := repos.Roles().GetByCodigo(ctx, strings.TrimSpace(*in.Rol))
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return &ValidationError{Msg: "rol inexistente"}
				}
				return err
			}
			if !rol.Activo {
				return &ValidationError{Msg: "rol inactivo"}
			}
			newRoleID = rol.ID
			newRoleCode = rol.Codigo
		}

		newDepartmentID := p.DepartamentoID
		if in.DepartamentoID != nil {
			departamentoID, _ := uuid.Parse(strings.TrimSpace(*in.DepartamentoID))
			departamento, err := repos.Departamentos().GetByID(ctx, departamentoID)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return &ValidationError{Msg: "departamento inexistente"}
				}
				return err
			}
			if !departamento.Activo {
				return &ValidationError{Msg: "departamento inactivo"}
			}
			newDepartmentID = departamento.ID
		}

		newState := p.Estado
		if in.Estado != nil {
			newState = *in.Estado
		}

		currentAdmin, err := s.roleHasPermission(ctx, repos, p.RolID, adminPermission)
		if err != nil {
			return err
		}
		newAdmin, err := s.roleHasPermission(ctx, repos, newRoleID, adminPermission)
		if err != nil {
			return err
		}
		currentAdmin = p.Estado == models.PersonalActivo && currentAdmin
		newAdmin = newState == models.PersonalActivo && newAdmin
		if currentAdmin && !newAdmin {
			activeAdmins, err := repos.Personal().LockActivosConPermiso(ctx, adminPermission)
			if err != nil {
				return err
			}
			others := 0
			for _, id := range activeAdmins {
				if id != targetID {
					others++
				}
			}
			if others == 0 {
				return ErrLastAdmin
			}
		}

		p.RolID = newRoleID
		p.DepartamentoID = newDepartmentID
		p.Estado = newState
		if in.Rol != nil || in.DepartamentoID != nil {
			p.VersionPermisos++
		}
		if newState == models.PersonalActivo {
			p.FallosConsecutivos = 0
			p.BloqueadoHasta = nil
		}

		if err := repos.Personal().Update(ctx, p, expectedVersion); err != nil {
			if errors.Is(err, repository.ErrVersionMismatch) {
				return ErrVersionMismatch
			}
			return err
		}
		if in.Estado != nil {
			if err := repos.Actores().SetActivo(ctx, targetID, newState == models.PersonalActivo); err != nil {
				return err
			}
		}
		if _, err := repos.Sesiones().RevocarTodasDeTitular(ctx, targetID, s.now()); err != nil {
			return err
		}
		if err := audit.Record(ctx, repos, &actor.ActorID, "users.update", models.ResultadoExito, &targetID, actor.RequestID, actor.IPOrigen); err != nil {
			return err
		}

		p.Version = expectedVersion + 1
		updated = p
		rolCode = newRoleCode
		return nil
	})
	if errors.Is(err, ErrLastAdmin) {
		s.auditDenied(ctx, actor, "users.update", "personal", &targetID)
	}
	if err != nil {
		return nil, err
	}
	return s.summary(updated, rolCode), nil
}

func (s *service) RevokeAllSessions(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string) error {
	if err := validateMotivo(motivo); err != nil {
		return err
	}
	if s.keyring == nil {
		return ErrCryptoUnavailable
	}

	return s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		if _, err := repos.Personal().GetByID(ctx, targetID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}

		revoked, err := repos.Sesiones().RevocarTodasDeTitular(ctx, targetID, s.now())
		if err != nil {
			return err
		}
		if err := s.recordRevocationOperation(ctx, repos, actor, targetID, motivo, revoked); err != nil {
			return err
		}
		return audit.Record(ctx, repos, &actor.ActorID, "users.sessions.revoke", models.ResultadoExito, &targetID, actor.RequestID, actor.IPOrigen)
	})
}

func (s *service) ResetMFA(ctx context.Context, actor repository.ActorContext, targetID uuid.UUID, motivo string, verificacionPresencial bool) (uuid.UUID, error) {
	if !verificacionPresencial {
		s.auditDenied(ctx, actor, "users.mfa.reset", "personal", &targetID)
		return uuid.Nil, ErrMFAVerificationRequired
	}
	if err := validateMotivo(motivo); err != nil {
		return uuid.Nil, err
	}
	if s.keyring == nil {
		return uuid.Nil, ErrCryptoUnavailable
	}

	var resetID uuid.UUID
	err := s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		if _, err := repos.Personal().GetByID(ctx, targetID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}

		resetID = uuid.New()
		payload, err := json.Marshal(map[string]any{
			"motivo":                 strings.TrimSpace(motivo),
			"verificacionPresencial": true,
		})
		if err != nil {
			return err
		}
		aad := []byte("identidad.recuperaciones_mfa:motivo:" + resetID.String())
		claveID, ciphertext, err := s.keyring.Seal(ctx, repos.ClavesRegistro(), "identidad", "recuperacion_mfa", resetID, payload, aad)
		if err != nil {
			return err
		}

		now := s.now()
		recuperacion := &models.RecuperacionMFA{
			ID:              resetID,
			PersonalID:      targetID,
			AdministradorID: actor.ActorID,
			VerificadoEn:    now,
			MotivoCifrado:   ciphertext,
			CompletadoEn:    &now,
			ClaveID:         claveID,
		}
		if err := repos.RecuperacionesMFA().Create(ctx, recuperacion); err != nil {
			return err
		}

		if err := repos.MFA().DeletePendiente(ctx, targetID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if err := repos.MFA().Invalidar(ctx, targetID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if _, err := repos.Sesiones().RevocarTodasDeTitular(ctx, targetID, now); err != nil {
			return err
		}

		return audit.Record(ctx, repos, &actor.ActorID, "users.mfa.reset", models.ResultadoExito, &targetID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return resetID, nil
}

func (s *service) recordRevocationOperation(
	ctx context.Context,
	repos repository.Repositories,
	actor repository.ActorContext,
	targetID uuid.UUID,
	motivo string,
	revoked int64,
) error {
	operationID := uuid.New()
	trimmedMotivo := strings.TrimSpace(motivo)
	payload, err := json.Marshal(map[string]any{
		"motivo":            trimmedMotivo,
		"sesionesRevocadas": revoked,
	})
	if err != nil {
		return err
	}
	aad := []byte("seguridad.operaciones:resultado_cifrado:" + operationID.String())
	claveID, ciphertext, err := s.keyring.Seal(ctx, repos.ClavesRegistro(), "seguridad", "operacion", operationID, payload, aad)
	if err != nil {
		return err
	}

	now := s.now()
	operacion := &models.Operacion{
		ID:               operationID,
		ActorID:          actor.ActorID,
		TipoOperacion:    "users.sessions.revoke",
		OperationID:      uuid.New(),
		RecursoAmbito:    &targetID,
		PeticionHMAC:     security.HMAC(s.secret, "users.sessions.revoke", actor.ActorID.String(), targetID.String(), trimmedMotivo),
		Estado:           models.OperacionAplicada,
		HTTPStatus:       204,
		ResultadoCifrado: ciphertext,
		ResultadoVenceEn: now.Add(justificationLifetime),
		AplicadaEn:       now,
		ClaveID:          claveID,
	}
	return repos.Operaciones().Create(ctx, operacion)
}

func (s *service) roleHasPermission(ctx context.Context, repos repository.Repositories, rolID uuid.UUID, codigo string) (bool, error) {
	permisos, err := repos.Permisos().ListByRol(ctx, rolID)
	if err != nil {
		return false, fmt.Errorf("usuario: permisos del rol %s: %w", rolID, err)
	}
	for _, permiso := range permisos {
		if permiso.Codigo == codigo {
			return true, nil
		}
	}
	return false, nil
}

func (s *service) summary(p *models.Personal, rolCode string) *usermodels.StaffSummary {
	return &usermodels.StaffSummary{
		ID:             p.ID,
		Usuario:        p.Usuario,
		NombreLaboral:  p.NombreLaboral,
		Rol:            rolCode,
		DepartamentoID: p.DepartamentoID,
		Estado:         p.Estado,
		VersionTag:     httpx.EncodeVersionTag(s.secret, p.Version),
	}
}

func (s *service) auditDenied(ctx context.Context, actor repository.ActorContext, action, tipoRecurso string, recursoID *uuid.UUID) {
	// Se escribe en una transacción nueva, ya cerrada la de negocio, para que la
	// denegación quede registrada aunque la operación haya hecho rollback.
	err := s.uow.Do(ctx, actor, func(ctx context.Context, repos repository.Repositories) error {
		return audit.RecordResource(ctx, repos, &actor.ActorID, action, tipoRecurso, models.ResultadoDenegado, recursoID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		log.Printf("usuario: auditoría de denegación %s: %v", action, err)
	}
}

func neutralDeviceLabel(dispositivo string) string {
	lower := strings.ToLower(dispositivo)
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "android") || strings.Contains(lower, "iphone") {
		return "Dispositivo móvil"
	}
	return "Navegador web"
}
