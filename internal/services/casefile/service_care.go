package casefile

import (
	"context"
	"errors"
	"strings"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

// ---------------- API-023 ----------------

func (s *service) Timeline(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, q casemodels.TimelineQuery, page repository.PageRequest) (*casemodels.Page[casemodels.TimelineItem], error) {
	for _, t := range q.Tipos {
		if t != "nota" && t != "cambio_estado" && t != "canalizacion" {
			return nil, vErr("tipo de timeline inv\u00e1lido")
		}
	}
	filterHash := httpx.FilterHash(recordID.String(), strings.Join(q.Tipos, ","), fmtTime(q.Desde), fmtTime(q.Hasta))
	plain, err := httpx.DecodeCursor(s.secret, page.Cursor, actor.ActorID, cursorScopeTimeline, filterHash)
	if err != nil {
		return nil, vErr("cursor inv\u00e1lido")
	}

	var result casemodels.Page[casemodels.TimelineItem]
	err = s.do(ctx, actor, "casefile.timeline", &recordID, func(ctx context.Context, r repository.Repositories) error {
		if _, err := loadExp(ctx, r, recordID, false); err != nil {
			return err
		}
		deps, err := r.Asignaciones().DepartamentosAsignados(ctx, actor.ActorID, recordID)
		if err != nil {
			return err
		}
		if len(deps) == 0 {
			return ErrNotFound
		}
		f := repository.TimelineFilter{DepartamentoIDs: deps, Tipos: q.Tipos}
		if q.Desde != nil {
			f.Rango.Desde = *q.Desde
		}
		if q.Hasta != nil {
			f.Rango.Hasta = *q.Hasta
		}
		rp, err := r.Timeline().ListAutorizado(ctx, recordID, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		result.Items = make([]casemodels.TimelineItem, 0, len(rp.Items))
		for _, it := range rp.Items {
			result.Items = append(result.Items, casemodels.TimelineItem{ID: it.ItemID, Tipo: it.Tipo,
				DepartamentoID: it.DepartamentoID, ActorID: it.ActorID, Resumen: it.Resumen, OcurridoEn: it.OcurridoEn.UTC()})
		}
		result.NextCursor = s.nextCursor(actor.ActorID, cursorScopeTimeline, filterHash, rp.NextCursor)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func fmtTime(t interface{ IsZero() bool }) string { return "" } // reemplazado abajo

// ---------------- API-028 ----------------

// nombresCampo son los nombres visibles de los campos revelables (bitácora).
var nombresCampo = map[string]string{"nombre": "nombre", "curp": "CURP", "fechaNacimiento": "fecha de nacimiento",
	"telefono": "teléfono", "domicilio": "domicilio"}

func (s *service) Disclose(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, in casemodels.DisclosureRequest) (*casemodels.DisclosureResponse, error) {
	campo, motivo := in.Campo, strings.TrimSpace(in.Motivo)
	idx := -1
	for i, f := range personaFields {
		if f == campo {
			idx = i
		}
	}
	fields := map[string]string{}
	if idx < 0 {
		fields["campo"] = "valor_invalido"
	}
	if !between(motivo, 10, 500) {
		fields["motivo"] = "motivo_required"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("revelación inválida", fields)
	}
	var out casemodels.DisclosureResponse
	err := s.do(ctx, actor, "casefile.disclose", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		p, err := r.Personas().GetByID(ctx, e.PersonaID)
		if err != nil {
			return err
		}
		payloads := [][]byte{p.NombreCifrado, p.CURPCifrada, p.FechaNacimientoCifrada, p.TelefonoCifrado, p.DomicilioCifrado}
		valor := ""
		if len(payloads[idx]) > 0 {
			valor, err = s.open(ctx, r, p.ClaveID, payloads[idx], "persona", campo, p.ID)
			if err != nil {
				return err
			}
		}
		// La auditor\u00eda va en la misma transacci\u00f3n: sin evento no hay revelaci\u00f3n.
		detalle := "Reveló " + nombresCampo[campo] + ". Motivo: " + motivo
		if err := audit.RecordDetail(ctx, r, &actor.ActorID, "casefile.disclose."+campo, "expediente",
			models.ResultadoExito, &e.ID, actor.RequestID, actor.IPOrigen, detalle); err != nil {
			return err
		}
		out = casemodels.DisclosureResponse{Campo: campo, Valor: valor}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------- API-024 / API-025 ----------------

func (s *service) Assign(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, in casemodels.AssignRequest) (*casemodels.AssignmentView, error) {
	personalID, err := parseUUID(in.PersonalID, "personalId")
	if err != nil {
		return nil, err
	}
	depID, err := parseUUID(in.DepartamentoID, "departamentoId")
	if err != nil {
		return nil, err
	}
	if !between(in.Motivo, 10, 500) {
		return nil, vErr("motivo debe tener entre 10 y 500 caracteres")
	}
	var solicitudID *uuid.UUID
	if in.SolicitudID != nil {
		id, err := parseUUID(*in.SolicitudID, "solicitudId")
		if err != nil {
			return nil, err
		}
		solicitudID = &id
	}
	var out casemodels.AssignmentView
	err = s.do(ctx, actor, "casefile.assign", &recordID, func(ctx context.Context, r repository.Repositories) error {
		// Quien asigna (Coordinación) no necesita ni obtiene acceso al contenido.
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if e.Estado == models.ExpedienteCerrado {
			return ErrCaseClosed
		}
		var sol *models.SolicitudVinculacion
		if solicitudID != nil {
			sol, err = r.SolicitudesVinculacion().GetByID(ctx, *solicitudID)
			if errors.Is(err, repository.ErrNotFound) {
				return vErr("solicitudId inexistente")
			}
			if err != nil {
				return err
			}
			if sol.ExpedienteID != e.ID {
				return vErr("la solicitud no corresponde a este expediente")
			}
			if sol.Estado != models.SolicitudPendiente {
				return ErrRequestNotPending
			}
		}
		target, err := r.Personal().GetByID(ctx, personalID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return vErr("personal inexistente")
			}
			return err
		}
		if target.Estado != models.PersonalActivo {
			return vErr("el personal no est\u00e1 activo")
		}
		if target.DepartamentoID != depID {
			return vErr("el personal no pertenece al departamento indicado")
		}
		a, err := s.assignTx(ctx, r, e.ID, personalID, depID, actor.ActorID, strings.TrimSpace(in.Motivo), s.now())
		if err != nil {
			return err
		}
		if sol != nil {
			if err := r.SolicitudesVinculacion().Resolver(ctx, sol.ID, models.SolicitudAprobada, actor.ActorID, s.now(), &a.ID, sol.Version); err != nil {
				if errors.Is(err, repository.ErrVersionMismatch) {
					return ErrRequestNotPending
				}
				return err
			}
			if err := s.audit(ctx, r, actor, "case_access_request.approve", "solicitud_vinculacion", &sol.ID); err != nil {
				return err
			}
		}
		out = casemodels.AssignmentView{ID: a.ID, PersonalID: a.PersonalID, DepartamentoID: a.DepartamentoID, Inicio: a.Inicio.UTC()}
		return s.audit(ctx, r, actor, "casefile.assign", "expediente", &e.ID)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) RevokeAssignment(ctx context.Context, actor repository.ActorContext, recordID, assignmentID uuid.UUID) error {
	return s.do(ctx, actor, "casefile.assign.revoke", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, true)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		a, err := r.Asignaciones().GetByID(ctx, assignmentID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		if a.ExpedienteID != e.ID {
			return ErrNotFound
		}
		if a.Fin != nil {
			return ErrAlreadyRevoked
		}
		vigentes, err := r.Asignaciones().ListByExpediente(ctx, e.ID, true)
		if err != nil {
			return err
		}
		if len(vigentes) <= 1 {
			return ErrLastAssignment
		}
		if err := r.Asignaciones().Cerrar(ctx, a.ID, s.now()); err != nil {
			return err
		}
		return s.audit(ctx, r, actor, "casefile.assign.revoke", "expediente", &e.ID)
	})
}

// ---------------- API-026 / API-027 ----------------

// maxNota es el largo máximo del texto de una nota (§6.2).
const maxNota = 10000

func (s *service) noteView(ctx context.Context, r repository.Repositories, n *models.Nota) (casemodels.NoteView, error) {
	texto, err := s.open(ctx, r, n.ClaveID, n.TextoCifrado, "nota", "texto", n.ID)
	if err != nil {
		return casemodels.NoteView{}, err
	}
	autor := ""
	if p, err := r.Personal().GetByID(ctx, n.AutorID); err == nil {
		autor = p.NombreLaboral
	} else if !errors.Is(err, repository.ErrNotFound) {
		return casemodels.NoteView{}, err
	}
	return casemodels.NoteView{ID: n.ID, EpisodioID: n.EpisodioID, AutorID: n.AutorID, Autor: autor, Texto: texto,
		OcurridoEn: n.OcurridoEn.UTC(), CorrigeNotaID: n.CorrigeNotaID, CreatedAt: n.CreadoEn.UTC()}, nil
}

func (s *service) ListNotes(ctx context.Context, actor repository.ActorContext, recordID, depID uuid.UUID, episodeID *uuid.UUID, page repository.PageRequest) (*casemodels.Page[casemodels.NoteView], error) {
	ep := ""
	if episodeID != nil {
		ep = episodeID.String()
	}
	filterHash := httpx.FilterHash(recordID.String(), depID.String(), ep)
	plain, err := httpx.DecodeCursor(s.secret, page.Cursor, actor.ActorID, cursorScopeNotes, filterHash)
	if err != nil {
		return nil, vErr("cursor inv\u00e1lido")
	}
	var result casemodels.Page[casemodels.NoteView]
	err = s.do(ctx, actor, "casefile.notes.list", &recordID, func(ctx context.Context, r repository.Repositories) error {
		if _, err := loadExp(ctx, r, recordID, false); err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, recordID, &depID); err != nil {
			return err
		}
		rp, err := r.Notas().ListBySeccion(ctx, recordID, depID, repository.NotaFilter{EpisodioID: episodeID},
			repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		result.Items = make([]casemodels.NoteView, 0, len(rp.Items))
		for i := range rp.Items {
			v, err := s.noteView(ctx, r, &rp.Items[i])
			if err != nil {
				return err
			}
			result.Items = append(result.Items, v)
		}
		result.NextCursor = s.nextCursor(actor.ActorID, cursorScopeNotes, filterHash, rp.NextCursor)
		return s.audit(ctx, r, actor, "casefile.notes.list", "expediente", &recordID)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) CreateNote(ctx context.Context, actor repository.ActorContext, recordID, depID, opID uuid.UUID, in casemodels.NoteCreateRequest) (*casemodels.NoteView, error) {
	if !between(in.Texto, 1, maxNota) {
		return nil, httpx.Invalid("texto inválido", map[string]string{"texto": "longitud_invalida"})
	}
	if in.OcurridoEn.IsZero() {
		return nil, vErr("ocurridoEn es obligatorio")
	}
	var corrige *uuid.UUID
	if in.CorrigeNotaID != nil {
		id, err := parseUUID(*in.CorrigeNotaID, "corrigeNotaId")
		if err != nil {
			return nil, err
		}
		corrige = &id
	}
	var out casemodels.NoteView
	err := s.do(ctx, actor, "casefile.notes.create", &recordID, func(ctx context.Context, r repository.Repositories) error {
		if prev, err := r.Notas().GetByOperacionID(ctx, opID); err == nil {
			out, err = s.noteView(ctx, r, prev)
			return err
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, &depID); err != nil {
			return err
		}
		if e.Estado == models.ExpedienteCerrado {
			return ErrCaseClosed
		}
		ep, err := r.Episodios().GetActivo(ctx, e.ID)
		if err != nil {
			return err
		}
		if corrige != nil {
			ok, err := r.Notas().EsPrevia(ctx, *corrige, e.ID, depID)
			if err != nil {
				return err
			}
			if !ok {
				return vErr("corrigeNotaId no corresponde a una nota previa de esta secci\u00f3n")
			}
			// Las notas son inmutables y cada una admite una sola corrección.
			corregida, err := r.Notas().TieneCorreccion(ctx, *corrige)
			if err != nil {
				return err
			}
			if corregida {
				return ErrAlreadyCorrected
			}
		}
		id := uuid.New()
		clave, payload, err := s.seal(ctx, r, "nota", "texto", id, strings.TrimSpace(in.Texto))
		if err != nil {
			return err
		}
		n := &models.Nota{ID: id, ExpedienteID: e.ID, EpisodioID: ep.ID, DepartamentoID: depID, AutorID: actor.ActorID,
			TextoCifrado: payload, OcurridoEn: in.OcurridoEn, CorrigeNotaID: corrige, OperacionID: opID, ClaveID: clave}
		if err := r.Notas().Create(ctx, n); err != nil {
			return err
		}
		n.CreadoEn = s.now()
		autor := ""
		if p, err := r.Personal().GetByID(ctx, actor.ActorID); err == nil {
			autor = p.NombreLaboral
		}
		out = casemodels.NoteView{ID: n.ID, EpisodioID: n.EpisodioID, AutorID: n.AutorID, Autor: autor, Texto: strings.TrimSpace(in.Texto),
			OcurridoEn: n.OcurridoEn.UTC(), CorrigeNotaID: corrige, CreatedAt: n.CreadoEn.UTC()}
		return s.audit(ctx, r, actor, "casefile.notes.create", "nota", &n.ID)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------- API-029 / API-030 / API-031 ----------------

func consentViews(ctx context.Context, r repository.Repositories, personaID uuid.UUID) ([]casemodels.ConsentView, error) {
	rows, err := r.EstadoPropositos().ListByPersona(ctx, personaID)
	if err != nil {
		return nil, err
	}
	out := make([]casemodels.ConsentView, 0, len(rows))
	for _, x := range rows {
		out = append(out, casemodels.ConsentView{ConsentID: x.DecisionID, PropositoID: x.PropositoID, Decision: x.Decision,
			Vigente: x.Vigente, Revocado: x.Revocado, DecididoEn: x.DecididoEn.UTC(), RevocadoEn: x.RevocadoEn})
	}
	return out, nil
}

func (s *service) ListConsents(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID) ([]casemodels.ConsentView, error) {
	var out []casemodels.ConsentView
	err := s.do(ctx, actor, "casefile.consents.list", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		out, err = consentViews(ctx, r, e.PersonaID)
		return err
	})
	return out, err
}

func (s *service) RecordConsent(ctx context.Context, actor repository.ActorContext, recordID uuid.UUID, opID *uuid.UUID, in casemodels.ConsentInput) ([]casemodels.ConsentView, error) {
	ids, err := validateConsentInput(in)
	if err != nil {
		return nil, err
	}
	var out []casemodels.ConsentView
	err = s.do(ctx, actor, "casefile.consents.create", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		if opID != nil {
			if _, err := r.Decisiones().GetByOperacionID(ctx, *opID); err == nil {
				out, err = consentViews(ctx, r, e.PersonaID)
				return err
			} else if !errors.Is(err, repository.ErrNotFound) {
				return err
			}
		}
		aviso, err := r.AvisosPrivacidad().GetVigente(ctx, s.now())
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNoPrivacyNotice
			}
			return err
		}
		if err := s.checkPurposes(ctx, r, aviso.ID, ids, false); err != nil {
			return err
		}
		pc, err := personaClave(ctx, r, e.PersonaID)
		if err != nil {
			return err
		}
		if err := s.recordDecision(ctx, r, e.PersonaID, pc, aviso.ID, actor.ActorID, in, ids, opID); err != nil {
			return err
		}
		if err := s.audit(ctx, r, actor, "casefile.consents.create", "expediente", &e.ID); err != nil {
			return err
		}
		out, err = consentViews(ctx, r, e.PersonaID)
		return err
	})
	return out, err
}

func (s *service) RevokeConsent(ctx context.Context, actor repository.ActorContext, recordID, consentID uuid.UUID, in casemodels.RevokeConsentRequest) ([]casemodels.ConsentView, error) {
	ids, err := parseUUIDs(in.PropositoIDs, 1, 10, "propositoIds")
	if err != nil {
		return nil, err
	}
	if in.Motivo != nil && len([]rune(*in.Motivo)) > 500 {
		return nil, vErr("motivo demasiado largo")
	}
	var out []casemodels.ConsentView
	err = s.do(ctx, actor, "casefile.consents.revoke", &recordID, func(ctx context.Context, r repository.Repositories) error {
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if err := access(ctx, r, actor.ActorID, e.ID, nil); err != nil {
			return err
		}
		d, err := r.Decisiones().GetByID(ctx, consentID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		if d.PersonaID != e.PersonaID {
			return ErrNotFound
		}
		if d.Decision != models.DecisionAceptado {
			return ErrNotRevocable
		}
		dps, err := r.DecisionPropositos().ListByDecision(ctx, d.ID)
		if err != nil {
			return err
		}
		byProp := make(map[uuid.UUID]uuid.UUID, len(dps))
		for _, dp := range dps {
			byProp[dp.PropositoID] = dp.ID
		}
		revID := uuid.New()
		items := make([]models.RevocacionProposito, 0, len(ids))
		for _, pid := range ids {
			dpID, ok := byProp[pid]
			if !ok {
				return vErr("una finalidad no pertenece a esa decisi\u00f3n")
			}
			prop, err := r.Propositos().GetByID(ctx, pid)
			if err != nil {
				return err
			}
			if !prop.Opcional || !prop.Revocable {
				return ErrNotRevocable
			}
			done, err := r.RevocacionPropositos().ExistePorDecisionProposito(ctx, dpID)
			if err != nil {
				return err
			}
			if done {
				return ErrAlreadyRevoked
			}
			items = append(items, models.RevocacionProposito{RevocacionID: revID, DecisionPropositoID: dpID})
		}
		pc, err := personaClave(ctx, r, e.PersonaID)
		if err != nil {
			return err
		}
		clave, motivo, err := s.sealOpt(ctx, r, "revocacion", "motivo", revID, in.Motivo, pc)
		if err != nil {
			return err
		}
		if err := r.Revocaciones().Create(ctx, &models.Revocacion{ID: revID, PersonaID: e.PersonaID, ActorID: actor.ActorID,
			MotivoCifrado: motivo, OcurridoEn: s.now(), ClaveID: clave}); err != nil {
			return err
		}
		if err := r.RevocacionPropositos().CreateMany(ctx, items); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return ErrAlreadyRevoked
			}
			return err
		}
		if err := s.audit(ctx, r, actor, "casefile.consents.revoke", "expediente", &e.ID); err != nil {
			return err
		}
		out, err = consentViews(ctx, r, e.PersonaID)
		return err
	})
	return out, err
}

// ---------------- API-033 ----------------

func (s *service) CreateReferral(ctx context.Context, actor repository.ActorContext, recordID, opID uuid.UUID, in casemodels.ReferralCreateRequest) (*casemodels.ReferralView, error) {
	destino, err := parseUUID(in.DestinoID, "destinoId")
	if err != nil {
		return nil, err
	}
	if !between(in.Resumen, 1, 2000) {
		return nil, vErr("resumen debe tener entre 1 y 2000 caracteres")
	}
	view := func(c *models.CanalizacionInterna) casemodels.ReferralView {
		return casemodels.ReferralView{ID: c.ID, ExpedienteID: c.ExpedienteID, OrigenID: c.OrigenID, DestinoID: c.DestinoID,
			Estado: c.Estado, VersionTag: httpx.EncodeVersionTag(s.secret, c.Version), CreatedAt: c.CreadoEn.UTC()}
	}
	var out casemodels.ReferralView
	err = s.do(ctx, actor, "casefile.referrals.create", &recordID, func(ctx context.Context, r repository.Repositories) error {
		if prev, err := r.Canalizaciones().GetByOperacionID(ctx, opID); err == nil {
			out = view(prev)
			return nil
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		e, err := loadExp(ctx, r, recordID, false)
		if err != nil {
			return err
		}
		if e.Estado == models.ExpedienteCerrado {
			return ErrCaseClosed
		}
		personal, err := r.Personal().GetByID(ctx, actor.ActorID)
		if err != nil {
			return err
		}
		origen := personal.DepartamentoID
		if err := access(ctx, r, actor.ActorID, e.ID, &origen); err != nil {
			return err
		}
		if origen == destino {
			return vErr("origen y destino no pueden ser el mismo departamento")
		}
		dep, err := r.Departamentos().GetByID(ctx, destino)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return vErr("departamento destino inexistente")
			}
			return err
		}
		if !dep.Activo {
			return vErr("departamento destino inactivo")
		}
		existe, err := r.Canalizaciones().ExisteActiva(ctx, e.ID, origen, destino)
		if err != nil {
			return err
		}
		if existe {
			return ErrReferralExists
		}
		id := uuid.New()
		clave, payload, err := s.seal(ctx, r, "canalizacion", "resumen", id, strings.TrimSpace(in.Resumen))
		if err != nil {
			return err
		}
		now := s.now()
		c := &models.CanalizacionInterna{ID: id, ExpedienteID: e.ID, OrigenID: origen, DestinoID: destino,
			EmitidaPor: actor.ActorID, ResumenCifrado: payload, Estado: models.CanalizacionPendiente,
			OperacionID: opID, ClaveID: clave, Version: 1, CreadoEn: now, ActualizadoEn: now}
		if err := r.Canalizaciones().Create(ctx, c); err != nil {
			return err
		}
		if err := r.HistorialCanalizaciones().Append(ctx, &models.HistorialCanalizacion{CanalizacionID: id, ActorID: actor.ActorID,
			EstadoNuevo: models.CanalizacionPendiente, VersionResultante: 1, ClaveID: clave}); err != nil {
			return err
		}
		out = view(c)
		return s.audit(ctx, r, actor, "casefile.referrals.create", "canalizacion", &id)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
