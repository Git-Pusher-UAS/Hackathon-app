// Package casefile implementa las reglas de negocio de /api/case-files.
package casefile

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const (
	modulo                = "expedientes"
	curpHMACVersion       = 1
	defaultTipoCasoCodigo = "por_clasificar"

	cursorScopeList     = "casefiles.list"
	cursorScopeNotes    = "casefiles.notes"
	cursorScopeTimeline = "casefiles.timeline"
)

var personaFields = []string{"nombre", "curp", "fechaNacimiento", "telefono", "domicilio"}

type Service interface {
	DuplicateCheck(ctx context.Context, actor repository.ActorContext, curp string) (bool, error)
	Create(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in casemodels.CreateRequest) (*casemodels.Summary, error)
	List(ctx context.Context, actor repository.ActorContext, q casemodels.ListQuery, page repository.PageRequest) (*casemodels.Page[casemodels.Summary], error)
	Get(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID) (*casemodels.Summary, error)
	Update(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, in casemodels.UpdateRequest, ifMatch string) (*casemodels.Summary, error)
	Timeline(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, q casemodels.TimelineQuery, page repository.PageRequest) (*casemodels.Page[casemodels.TimelineItem], error)
	Disclose(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, campo string) (*casemodels.DisclosureResponse, error)
	Assign(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, in casemodels.AssignRequest) (*casemodels.AssignmentView, error)
	RevokeAssignment(ctx context.Context, actor repository.ActorContext, recordID, assignmentID uuid.UUID) error
	ListNotes(ctx context.Context, actor repository.ActorContext, recordID, depID uuid.UUID, episodeID *uuid.UUID, page repository.PageRequest) (*casemodels.Page[casemodels.NoteView], error)
	CreateNote(ctx context.Context, actor repository.ActorContext, recordID, depID, opID uuid.UUID, in casemodels.NoteCreateRequest) (*casemodels.NoteView, error)
	ListConsents(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID) ([]casemodels.ConsentView, error)
	RecordConsent(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, opID *uuid.UUID, in casemodels.ConsentInput) ([]casemodels.ConsentView, error)
	RevokeConsent(ctx context.Context, actor repository.ActorContext, recordID, consentID uuid.UUID, in casemodels.RevokeConsentRequest) ([]casemodels.ConsentView, error)
	CreateReferral(ctx context.Context, actor repository.ActorContext, recordID, opID uuid.UUID, in casemodels.ReferralCreateRequest) (*casemodels.ReferralView, error)
}

type service struct {
	uow     repository.UnitOfWork
	keyring *security.Keyring
	secret  string // firma cursores y versionTags
	curpKey string // llave HMAC del \u00edndice ciego de CURP
	now     func() time.Time
}

func New(uow repository.UnitOfWork, keyring *security.Keyring, secret, curpKey string) *service {
	return &service{uow: uow, keyring: keyring, secret: secret, curpKey: curpKey, now: time.Now}
}

// ---------------- helpers compartidos ----------------

func aad(tipo, campo string, id uuid.UUID) []byte {
	return []byte(tipo + ":" + campo + ":" + id.String())
}

func optBytes(s *string) []byte {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return []byte(strings.TrimSpace(*s))
}

func (s *service) curpHMAC(curp string) []byte { return security.HMAC(s.curpKey, "curp", curp) }

// do ejecuta fn en transacci\u00f3n y audita la denegaci\u00f3n si el recurso no existe o no es accesible.
func (s *service) do(ctx context.Context, actor repository.ActorContext, accion string, recursoID *uuid.UUID, fn func(ctx context.Context, r repository.Repositories) error) error {
	err := s.uow.Do(ctx, actor, fn)
	if errors.Is(err, ErrNotFound) {
		_ = s.uow.DoIndependent(ctx, func(ctx context.Context, r repository.Repositories) error {
			return audit.RecordResource(ctx, r, &actor.ActorID, accion, "expediente", models.ResultadoDenegado, recursoID, actor.RequestID, actor.IPOrigen)
		})
	}
	return err
}

func (s *service) audit(ctx context.Context, r repository.Repositories, actor repository.ActorContext, accion, tipo string, id *uuid.UUID) error {
	return audit.RecordResource(ctx, r, &actor.ActorID, accion, tipo, models.ResultadoExito, id, actor.RequestID, actor.IPOrigen)
}

func access(ctx context.Context, r repository.Repositories, actorID, expID uuid.UUID, dep *uuid.UUID) error {
	ok, err := r.Asignaciones().TieneAcceso(ctx, actorID, expID, dep)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func loadExp(ctx context.Context, r repository.Repositories, id uuid.UUID, forUpdate bool) (*models.Expediente, error) {
	var e *models.Expediente
	var err error
	if forUpdate {
		e, err = r.Expedientes().GetByIDForUpdate(ctx, id)
	} else {
		e, err = r.Expedientes().GetByID(ctx, id)
	}
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return e, err
}

func (s *service) seal(ctx context.Context, r repository.Repositories, tipo, campo string, id uuid.UUID, text string) (uuid.UUID, []byte, error) {
	if s.keyring == nil {
		return uuid.Nil, nil, ErrCryptoUnavailable
	}
	return s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, tipo, id, []byte(text), aad(tipo, campo, id))
}

// sealOpt cifra solo si hay texto; si no, devuelve la clave de respaldo y payload nil.
func (s *service) sealOpt(ctx context.Context, r repository.Repositories, tipo, campo string, id uuid.UUID, text *string, fallback uuid.UUID) (uuid.UUID, []byte, error) {
	if text == nil || strings.TrimSpace(*text) == "" {
		return fallback, nil, nil
	}
	return s.seal(ctx, r, tipo, campo, id, strings.TrimSpace(*text))
}

func (s *service) open(ctx context.Context, r repository.Repositories, claveID uuid.UUID, payload []byte, tipo, campo string, id uuid.UUID) (string, error) {
	if s.keyring == nil {
		return "", ErrCryptoUnavailable
	}
	pt, err := s.keyring.Open(ctx, r.ClavesRegistro(), claveID, payload, aad(tipo, campo, id))
	return string(pt), err
}

func personaClave(ctx context.Context, r repository.Repositories, personaID uuid.UUID) (uuid.UUID, error) {
	p, err := r.Personas().GetByID(ctx, personaID)
	if err != nil {
		return uuid.Nil, err
	}
	return p.ClaveID, nil
}

func activeTipo(ctx context.Context, r repository.Repositories, expID uuid.UUID) (*uuid.UUID, error) {
	ep, err := r.Episodios().GetActivo(ctx, expID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ep.TipoCasoID, nil
}

func (s *service) summary(e *models.Expediente, tipo *uuid.UUID) casemodels.Summary {
	return casemodels.Summary{
		ID: e.ID, Alias: e.Alias, Estado: e.Estado, OrigenContacto: e.OrigenContacto, TipoCasoID: tipo,
		CreatedAt: e.CreadoEn.UTC(), UpdatedAt: e.ActualizadoEn.UTC(), ClosedAt: e.CerradoEn,
		VersionTag: httpx.EncodeVersionTag(s.secret, e.Version),
	}
}

func (s *service) nextCursor(actorID uuid.UUID, scope, filter, repoNext string) *string {
	if repoNext == "" {
		return nil
	}
	c := httpx.EncodeCursor(s.secret, actorID, scope, filter, repoNext)
	return &c
}

func (s *service) assignTx(ctx context.Context, r repository.Repositories, expID, personalID, depID, by uuid.UUID, motivo string, at time.Time) (*models.Asignacion, error) {
	id := uuid.New()
	claveID, payload, err := s.seal(ctx, r, "asignacion", "motivo", id, motivo)
	if err != nil {
		return nil, err
	}
	a := &models.Asignacion{ID: id, ExpedienteID: expID, PersonalID: personalID, DepartamentoID: depID,
		AsignadaPor: by, Inicio: at, MotivoCifrado: payload, ClaveID: claveID}
	if err := r.Asignaciones().Create(ctx, a); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAlreadyAssigned
		}
		return nil, err
	}
	return a, nil
}

func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func (s *service) checkPurposes(ctx context.Context, r repository.Repositories, avisoID uuid.UUID, ids []uuid.UUID, requireMandatory bool) error {
	ok, err := r.AvisoPropositos().AdmiteTodos(ctx, avisoID, ids)
	if err != nil {
		return err
	}
	if !ok {
		return vErr("alguna finalidad no pertenece al aviso de privacidad vigente")
	}
	if requireMandatory {
		props, err := r.Propositos().ListByAviso(ctx, avisoID)
		if err != nil {
			return err
		}
		for _, p := range props {
			if !p.Opcional && !contains(ids, p.ID) {
				return vErr("faltan finalidades obligatorias del aviso")
			}
		}
	}
	return nil
}

func (s *service) recordDecision(ctx context.Context, r repository.Repositories, personaID, claveID, avisoID, actorID uuid.UUID, in casemodels.ConsentInput, ids []uuid.UUID, opID *uuid.UUID) error {
	if _, err := r.Personas().GetByIDForUpdate(ctx, personaID); err != nil {
		return err
	}
	orden, err := r.Decisiones().SiguienteOrden(ctx, personaID)
	if err != nil {
		return err
	}
	d := &models.Decision{ID: uuid.New(), PersonaID: personaID, AvisoID: avisoID, ActorID: actorID,
		Decision: in.Decision, Modalidad: in.Modalidad, CapturadoEn: in.CapturadoEn,
		OrdenPersona: orden, ClaveID: claveID, OperacionID: opID}
	if err := r.Decisiones().Create(ctx, d); err != nil {
		return err
	}
	items := make([]models.DecisionProposito, 0, len(ids))
	for _, pid := range ids {
		items = append(items, models.DecisionProposito{DecisionID: d.ID, PropositoID: pid})
	}
	return r.DecisionPropositos().CreateMany(ctx, items)
}

// ---------------- API-016 ----------------

func (s *service) DuplicateCheck(ctx context.Context, actor repository.ActorContext, curp string) (bool, error) {
	if s.curpKey == "" {
		return false, ErrCryptoUnavailable
	}
	curp = normalizeCURP(curp)
	if !curpRe.MatchString(curp) {
		return false, vErr("curp inv\u00e1lida")
	}
	dup := false
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		_, err := r.IndicesCURP().BuscarPersona(ctx, []repository.IndiceCURPClave{{VersionHMAC: curpHMACVersion, ValorHMAC: s.curpHMAC(curp)}})
		if err == nil {
			dup = true
			return nil
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	})
	return dup, err
}

// ---------------- API-017 ----------------

func (s *service) Create(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in casemodels.CreateRequest) (*casemodels.Summary, error) {
	if s.keyring == nil || s.curpKey == "" {
		return nil, ErrCryptoUnavailable
	}
	v, err := validateCreate(in)
	if err != nil {
		return nil, err
	}

	var out casemodels.Summary
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		// Reintento idempotente: misma operaci\u00f3n => mismo expediente.
		if prev, err := r.Expedientes().GetByOperacionID(ctx, opID); err == nil {
			tipo, err := activeTipo(ctx, r, prev.ID)
			if err != nil {
				return err
			}
			out = s.summary(prev, tipo)
			return nil
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}

		now := s.now()
		personal, err := r.Personal().GetByID(ctx, actor.ActorID)
		if err != nil {
			return err
		}
		aviso, err := r.AvisosPrivacidad().GetVigente(ctx, now)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNoPrivacyNotice
			}
			return err
		}
		if err := s.checkPurposes(ctx, r, aviso.ID, v.propositoIDs, true); err != nil {
			return err
		}
		if _, err := r.Sectores().GetByID(ctx, v.sectorID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return vErr("sector inexistente")
			}
			return err
		}
		if v.instID != nil {
			if _, err := r.Instituciones().GetByID(ctx, *v.instID); err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return vErr("instituci\u00f3n inexistente")
				}
				return err
			}
		}
		var tipoID uuid.UUID
		if v.tipoCasoID != nil {
			t, err := r.TiposCaso().GetByID(ctx, *v.tipoCasoID)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return vErr("tipoCasoId inexistente")
				}
				return err
			}
			tipoID = t.ID
		} else {
			t, err := r.TiposCaso().GetByCodigo(ctx, defaultTipoCasoCodigo)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return vErr("falta el tipo de caso '" + defaultTipoCasoCodigo + "' en el cat\u00e1logo; env\u00eda tipoCasoId")
				}
				return err
			}
			tipoID = t.ID
		}

		// Duplicado por \u00edndice ciego de CURP.
		hm := s.curpHMAC(v.curp)
		if _, err := r.IndicesCURP().BuscarPersona(ctx, []repository.IndiceCURPClave{{VersionHMAC: curpHMACVersion, ValorHMAC: hm}}); err == nil {
			return ErrDuplicatePerson
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}

		// Persona: 5 campos, una DEK.
		personaID := uuid.New()
		plain := [][]byte{[]byte(v.nombre), []byte(v.curp), []byte(v.fecha.Format("2006-01-02")), optBytes(in.Telefono), optBytes(in.Domicilio)}
		aads := make([][]byte, len(personaFields))
		for i, f := range personaFields {
			aads[i] = aad("persona", f, personaID)
		}
		claveID, sealed, err := s.keyring.SealMany(ctx, r.ClavesRegistro(), modulo, "persona", personaID, plain, aads)
		if err != nil {
			return err
		}
		persona := &models.Persona{ID: personaID, NombreCifrado: sealed[0], CURPCifrada: sealed[1],
			FechaNacimientoCifrada: sealed[2], TelefonoCifrado: sealed[3], DomicilioCifrado: sealed[4],
			SectorID: v.sectorID, RegistradaPor: actor.ActorID, ClaveID: claveID, Version: 1,
			ActualizadoEn: now, CreadoEn: now}
		if err := r.Personas().Create(ctx, persona); err != nil {
			return err
		}
		if err := r.IndicesCURP().Create(ctx, &models.IndiceCURP{PersonaID: personaID, VersionHMAC: curpHMACVersion, ValorHMAC: hm}); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrDuplicatePerson
			}
			return err
		}

		exp := &models.Expediente{ID: uuid.New(), PersonaID: personaID, Alias: v.alias, Estado: models.ExpedienteAbierto,
			OrigenContacto: in.OrigenContacto, InstitucionOrigenID: v.instID, CreadoPor: actor.ActorID,
			Version: 1, ActualizadoEn: now, CreadoEn: now, OperacionID: opID}
		if err := r.Expedientes().Create(ctx, exp); err != nil {
			return err
		}
		if err := r.Episodios().Create(ctx, &models.Episodio{ExpedienteID: exp.ID, Numero: 1, TipoCasoID: tipoID, AbiertoEn: now}); err != nil {
			return err
		}
		if err := r.HistorialEstados().Append(ctx, &models.HistorialEstado{ExpedienteID: exp.ID, EstadoNuevo: models.ExpedienteAbierto,
			ActorID: actor.ActorID, VersionResultante: 1, ClaveID: claveID}); err != nil {
			return err
		}
		if _, err := s.assignTx(ctx, r, exp.ID, actor.ActorID, personal.DepartamentoID, actor.ActorID, "Alta del expediente", now); err != nil {
			return err
		}
		if err := s.recordDecision(ctx, r, personaID, claveID, aviso.ID, actor.ActorID, in.Consentimiento, v.propositoIDs, nil); err != nil {
			return err
		}
		if err := s.audit(ctx, r, actor, "casefile.create", "expediente", &exp.ID); err != nil {
			return err
		}
		out = s.summary(exp, &tipoID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------- API-020 / API-021 ----------------

func (s *service) List(ctx context.Context, actor repository.ActorContext, q casemodels.ListQuery, page repository.PageRequest) (*casemodels.Page[casemodels.Summary], error) {
	alias := strings.TrimSpace(q.Alias)
	filterHash := httpx.FilterHash(string(q.Estado), alias)
	plain, err := httpx.DecodeCursor(s.secret, page.Cursor, actor.ActorID, cursorScopeList, filterHash)
	if err != nil {
		return nil, vErr("cursor inv\u00e1lido")
	}
	f := repository.ExpedienteFilter{AsignadoA: &actor.ActorID, Alias: alias}
	if q.Estado != "" {
		est := q.Estado
		f.Estado = &est
	}

	var result casemodels.Page[casemodels.Summary]
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.Expedientes().List(ctx, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		result.Items = make([]casemodels.Summary, 0, len(rp.Items))
		for i := range rp.Items {
			tipo, err := activeTipo(ctx, r, rp.Items[i].ID)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, s.summary(&rp.Items[i], tipo))
		}
		result.NextCursor = s.nextCursor(actor.ActorID, cursorScopeList, filterHash, rp.NextCursor)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) Get(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID) (*casemodels.Summary, error) {
	var out casemodels.Summary
	err := s.do(ctx, actor, "casefile.get", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		tipo, err := activeTipo(ctx, r, e.ID)
		if err != nil {
			return err
		}
		out = s.summary(e, tipo)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------- API-022 ----------------

func (s *service) Update(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, in casemodels.UpdateRequest, ifMatch string) (*casemodels.Summary, error) {
	if err := validateUpdate(in); err != nil {
		return nil, err
	}
	if ifMatch == "" {
		return nil, ErrPreconditionRequired
	}
	expected, err := httpx.DecodeVersionTag(s.secret, ifMatch)
	if err != nil {
		return nil, ErrVersionMismatch
	}

	var out casemodels.Summary
	err = s.do(ctx, actor, "casefile.update", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, true)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		if e.Version != expected {
			return ErrVersionMismatch
		}
		now := s.now()
		prev := e.Estado

		var newTipo *uuid.UUID
		if in.TipoCasoID != nil {
			id, _ := uuid.Parse(strings.TrimSpace(*in.TipoCasoID))
			t, err := r.TiposCaso().GetByID(ctx, id)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return vErr("tipoCasoId inexistente")
				}
				return err
			}
			newTipo = &t.ID
		}
		if in.Alias != nil {
			e.Alias = strings.TrimSpace(*in.Alias)
		}

		changed := in.Estado != nil && *in.Estado != prev
		reopen := false
		if changed {
			if !transitionAllowed(prev, *in.Estado) {
				return ErrInvalidTransition
			}
			if prev == models.ExpedienteCerrado && (in.Motivo == nil || strings.TrimSpace(*in.Motivo) == "") {
				return vErr("motivo es obligatorio al reabrir")
			}
			e.Estado = *in.Estado
			switch {
			case e.Estado == models.ExpedienteCerrado:
				e.CerradoEn = &now
				ep, err := r.Episodios().GetActivo(ctx, e.ID)
				if err == nil {
					if err := r.Episodios().Cerrar(ctx, ep.ID, now); err != nil {
						return err
					}
				} else if !errors.Is(err, repository.ErrNotFound) {
					return err
				}
			case prev == models.ExpedienteCerrado:
				e.CerradoEn = nil
				reopen = true
			}
		} else if prev == models.ExpedienteCerrado && (in.Alias != nil || newTipo != nil) {
			return ErrCaseClosed
		}

		if newTipo != nil && !reopen {
			if ep, err := r.Episodios().GetActivo(ctx, e.ID); err == nil {
				if err := r.Episodios().ActualizarTipoCaso(ctx, ep.ID, *newTipo); err != nil {
					return err
				}
			} else if !errors.Is(err, repository.ErrNotFound) {
				return err
			}
		}

		if err := r.Expedientes().Update(ctx, e, expected); err != nil {
			if errors.Is(err, repository.ErrVersionMismatch) {
				return ErrVersionMismatch
			}
			return err
		}
		e.Version = expected + 1
		e.ActualizadoEn = now

		if changed {
			pc, err := personaClave(ctx, r, e.PersonaID)
			if err != nil {
				return err
			}
			hid := uuid.New()
			clave, motivo, err := s.sealOpt(ctx, r, "historial_estado", "motivo", hid, in.Motivo, pc)
			if err != nil {
				return err
			}
			if err := r.HistorialEstados().Append(ctx, &models.HistorialEstado{ID: hid, ExpedienteID: e.ID, EstadoAnterior: &prev,
				EstadoNuevo: e.Estado, ActorID: actor.ActorID, MotivoCifrado: motivo, VersionResultante: e.Version, ClaveID: clave}); err != nil {
				return err
			}
		}
		if reopen {
			eps, err := r.Episodios().ListByExpediente(ctx, e.ID)
			if err != nil {
				return err
			}
			tipo := uuid.Nil
			if len(eps) > 0 {
				tipo = eps[len(eps)-1].TipoCasoID
			}
			if newTipo != nil {
				tipo = *newTipo
			}
			num, err := r.Episodios().SiguienteNumero(ctx, e.ID)
			if err != nil {
				return err
			}
			if err := r.Episodios().Create(ctx, &models.Episodio{ExpedienteID: e.ID, Numero: num, TipoCasoID: tipo, AbiertoEn: now}); err != nil {
				return err
			}
		}
		if err := s.audit(ctx, r, actor, "casefile.update", "expediente", &e.ID); err != nil {
			return err
		}
		tipo, err := activeTipo(ctx, r, e.ID)
		if err != nil {
			return err
		}
		out = s.summary(e, tipo)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
