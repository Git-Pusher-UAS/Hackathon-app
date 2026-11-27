# Simulated Demo Seed Data (`cmd/seed-demo`)

> **Dev-only & idempotent:** `cmd/seed-demo` refuses to run unless `APP_ENV` is `dev`, `development`, or `local`. It can be executed multiple times without duplicating rows.

---

## How to Run

```bash
# 1. Create schemas, tables, and views
go run ./cmd/migrate

# 2. Bootstrap base roles, RBAC permissions, and initial admin
go run ./cmd/seed

# 3. Seed simulated demo data (requires APP_ENV=dev|development|local)
APP_ENV=dev go run ./cmd/seed-demo
```

> **Tip for local OTP testing:** Start the API with `APP_ENV=dev go run ./cmd` so 6-digit OTP codes for Portal and Employer logins print to server stdout (`[DEV OTP] ... code=XXXXXX`).

---

## 1. Staff Accounts (`identidad.personal`)

All demo staff accounts share password **`Demo1234!`**, require no password change, and have **no MFA enrolled** (login completes in one `POST /api/auth/login` request).

| Username | Password | Role | Department | Primary Flows to Test |
|---|---|---|---|---|
| `ts.demo` | `Demo1234!` | `TrabajoSocial` | `TRABAJO_SOCIAL` | Create/read/update case files, notes, consents, internal & external referrals, disclosures |
| `psi.demo` | `Demo1234!` | `Psicología` | `PSICOLOGIA` | Receive/accept pending internal referral on `Caso-Demo-01`, request access to other cases |
| `jur.demo` | `Demo1234!` | `Jurídico` | `JURIDICO` | Assigned to `Caso-Demo-02`, read notes/timeline, legal follow-up |
| `coord.demo` | `Demo1234!` | `Coordinación` | `TRABAJO_SOCIAL` | Assign cases, review access requests, view analytics & heatmap |
| `cap.demo` | `Demo1234!` | `Capacitación` | `CAPACITACION` | Create/list courses, mark attendance, manage jobs & admin bolsa, outreach visits, survey results |
| `dir.demo` | `Demo1234!` | `Dirección` | `DIRECCION` | Audit logs/alerts, analytics summary, heatmap, survey results |

---

## 2. Portal Usuaria Accounts (`portal.*`)

Login via `POST /api/portal/auth/login` using either **CURP** or **Nombre**, then read the 6-digit OTP from dev server logs and verify via `POST /api/portal/auth/verify`.

| ID | Alias | Nombre | CURP | Telefono | Trusted Contact | CV | Pre-seeded Activity |
|---|---|---|---|---|---|---|---|
| `11111111-1111-4111-8111-111111111101` | `Usuaria-Demo01` | `Ana Morales Demo` | `MOLA920514MSLRNS01` | `6671112233` | `Rosa Morales` (`6679990011`) | Complete (profile + 1 experience + 1 education + skills) | Enrolled in Course 1 (`inscrita: true`), applied to Job 1 (`submitted`), 1 pending survey |
| `11111111-1111-4111-8111-111111111102` | `Usuaria-Demo02` | `María Torres Demo` | `TOMA881123MSLRTN02` | `6672223344` | `Elena Torres` (`6678880022`) | None (tests CV-required validation on `POST /api/jobs/:id/applications`) | Not enrolled in any course (`inscrita: false`), 1 pending survey |

---

## 3. Employers & Job Board (`bolsa.*`)

Login via `POST /api/employers/auth/login` with **Correo**, then read the 6-digit OTP from dev server logs and verify via `POST /api/employers/auth/verify`.

### Employers

| ID | Nombre | Correo | Telefono | Estado | Notes |
|---|---|---|---|---|---|
| `22222222-2222-4222-8222-222222222201` | `Comercializadora Pacífico Demo` | `rrhh@empresa-demo.mx` | `6673334455` | `approved` | Owns 3 jobs (2 published, 1 pending review); has 1 application from `Usuaria-Demo01` |
| `22222222-2222-4222-8222-222222222202` | `Taller Textil Creativo Demo` | `contacto@taller-demo.mx` | `6674445566` | `pending_review` | Appears in staff `GET /api/admin/employers` review queue |

### Jobs (`bolsa.vacantes`)

| ID | Puesto | Sector | Horario | Tipo | Ubicación | Estado |
|---|---|---|---|---|---|---|
| `22222222-2222-4222-8222-222222222211` | `Auxiliar Administrativo` | `Administración` | `Matutino` | `Tiempo completo` | `Centro, Culiacán` | `published` (`Usuaria-Demo01` applied) |
| `22222222-2222-4222-8222-222222222212` | `Atención a Clientes` | `Comercio` | `Vespertino` | `Medio tiempo` | `Colonia Norte, Culiacán` | `published` |
| `22222222-2222-4222-8222-222222222213` | `Recepcionista` | `Servicios` | `Matutino` | `Tiempo completo` | `Centro, Culiacán` | `pending_review` (staff admin queue) |

---

## 4. Courses & Outreach Visits (`capacitacion.*`)

### Courses (`capacitacion.cursos`)

All course dates are seeded in the future relative to the seed run date (`+14d`, `+21d`, `+30d`).

| ID | Tipo | Título | Hora | Lugar | Cupo | Pre-enrolled |
|---|---|---|---|---|---|---|
| `33333333-3333-4333-8333-333333333301` | `curso` | `Finanzas Personales y Emprendimiento` | `10:00` | `Aula 1 - Centro` | `20` | `Usuaria-Demo01` (`Participante 1`) |
| `33333333-3333-4333-8333-333333333302` | `charla` | `Derechos Laborales y Seguridad Social` | `12:00` | `Auditorio Municipal` | `35` | — |
| `33333333-3333-4333-8333-333333333303` | `conferencia` | `Autonomía Económica y Liderazgo` | `16:00` | `Salón Principal` | `50` | — |

### Outreach Visits (`capacitacion.visitas`)

| ID | Sector | Fecha | Tema | Responsable | Estado |
|---|---|---|---|---|---|
| `33333333-3333-4333-8333-333333333311` | `SEC-CENTRO` | `+10d` | `Difusión de servicios y prevención en comercio local` | `Andrea Capacitación Demo` | `planeada` |
| `33333333-3333-4333-8333-333333333312` | `SEC-NORTE` | `-5d` | `Jornada informativa en colonia Norte` | `Andrea Capacitación Demo` | `realizada` |

---

## 5. Surveys (`encuestas.*`)

- **Survey ID:** `44444444-4444-4444-8444-444444444401`
- **Title:** `Encuesta de satisfacción de atención` (`3` min, `activa: true`)
- **Questions:**
  - `q1` (`escala`, 1–5): `¿Qué tan clara fue la atención recibida? (1 a 5)`
  - `q2` (`opcion`, `["Excelente","Buena","Regular"]`): `¿Cómo calificas el trato del personal?`
  - `q3` (`abierta`): `¿Tienes algún comentario o sugerencia?`
- **Assignments:** Pending (`contestada: false`) for both `Usuaria-Demo01` and `Usuaria-Demo02`.
- **K-anonymity (`K = 5`):** 5 anonymous submissions are pre-seeded (scores `5, 4, 5, 4, 5` -> avg `4.6`; option `"Excelente"` x5) so `GET /api/surveys/results` immediately returns non-suppressed aggregate metrics.

---

## 6. Case Files (`expedientes.*`) & Analytics (`K = 5`)

Created through the real encrypted `casefile.Service` (requires `KEK_MASTER` and `CURP_HMAC_KEY` in `.env`). All 5 cases are in `SEC-CENTRO` with `violencia_familiar` so `analitica.v_resumen_mvp` and `analitica.v_mapa_mvp` (`GET /api/analytics/summary`, `GET /api/analytics/heatmap`) satisfy `K >= 5` and return real non-null counts immediately.

| Alias | Nombre (Encrypted) | CURP (Encrypted + Blind Index) | DOB | Telefono | Origen | Estado | Extra Seeded Records |
|---|---|---|---|---|---|---|---|
| `Caso-Demo-01` | `Lucía Vega Demo` | `VELU900310MSLGVR01` | `1990-03-10` | `6675001001` | `presencial` | `abierto` | Assigned to `ts.demo`; 1 note; 1 **pending internal referral** (`TRABAJO_SOCIAL` -> `PSICOLOGIA`) |
| `Caso-Demo-02` | `Sofía Castro Demo` | `CASO870722MSLSTN02` | `1987-07-22` | `6675001002` | `linea_atencion` | `en_seguimiento` | Assigned to `ts.demo` & `jur.demo`; 1 note; 1 **external referral** (`Fiscalía Especializada en Atención a Mujeres`) |
| `Caso-Demo-03` | `Elena Ruiz Demo` | `RUEL950115MSLZRD03` | `1995-01-15` | `6675001003` | `presencial` | `abierto` | Assigned to `ts.demo`; 1 note |
| `Caso-Demo-04` | `Carmen Díaz Demo` | `DICA910908MSLZCN04` | `1991-09-08` | `6675001004` | `institucion` | `abierto` | Origin institution: `Línea de Atención a las Mujeres`; assigned to `ts.demo` |
| `Caso-Demo-05` | `Patricia Navarro Demo` | `NAPA840630MSLVRT05` | `1984-06-30` | `6675001005` | `patrulla_genero` | `abierto` | Assigned to `ts.demo` |

---

## 7. Seeded Catalogs (`catalogos.*`)

- **Departments:** `TRABAJO_SOCIAL`, `PSICOLOGIA`, `JURIDICO` (asistenciales), `CAPACITACION`, `DIRECCION`
- **Sectors:** `SEC-CENTRO` (`sector`), `SEC-NORTE` (`colonia`), `SEC-SUR` (`sindicatura`)
- **Case Types:** `por_clasificar`, `violencia_familiar`, `apoyo_psicologico`, `orientacion_juridica`
- **Institutional Contacts (`publicable: true`):**
  - `Línea de Emergencias 911` (`911`, prioritario)
  - `Línea de Atención a las Mujeres` (`800-822-7956`, prioritario)
  - `Fiscalía Especializada en Atención a Mujeres` (`667-713-3200`)
- **Privacy Notice:** `v1.0-demo` linked to `atencion_integral` (mandatory), `seguimiento_caso` (mandatory, revocable), `vinculacion_bolsa` (optional, revocable)
