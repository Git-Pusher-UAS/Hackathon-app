# Endpoint → Folder Mapping

Reference document mapping each API endpoint from `ENDPOINTS_SAI_MVP.md` to its corresponding folder under `internal/(handlers|middleware|models|services)/`.

**Base path:** `/api/v1`

---

## 1. `health/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-001 | `GET` | `/health/live` | API health check |

---

## 2. `auth/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-002 | `GET` | `/auth/csrf-token` | Get CSRF token |
| API-003 | `POST` | `/auth/login` | Validate credentials |
| API-004 | `POST` | `/auth/mfa/enrollments` | Start MFA enrollment |
| API-005 | `POST` | `/auth/mfa/verify` | Verify MFA code |
| API-007 | `POST` | `/auth/logout` | Logout current session |

---

## 3. `user/`

Includes user accounts and sessions management.

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-006 | `GET` | `/users/me` | Get current account |
| API-008 | `GET` | `/users/me/sessions` | List own sessions |
| API-009 | `DELETE` | `/users/me/sessions/{sessionId}` | Revoke own session |
| API-010 | `GET` | `/users` | List staff accounts |
| API-011 | `POST` | `/users` | Create staff account |
| API-012 | `PATCH` | `/users/{userId}` | Update account permissions/status |
| API-013 | `POST` | `/users/{userId}/session-revocations` | Revoke all sessions of an account |
| API-014 | `POST` | `/users/{userId}/mfa-resets` | Reset MFA for an account |

---

## 4. `catalog/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-015 | `GET` | `/catalogs` | Get catalogs (departments, roles, etc.) |

---

## 5. `casefile/`

The largest folder. Groups: case-file CRUD, assignments, department notes, consents, referral creation, disclosures, and timeline.

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-016 | `POST` | `/case-files/duplicate-checks` | Check for duplicate person |
| API-017 | `POST` | `/case-files` | Create case-file + initial consent |
| API-020 | `GET` | `/case-files` | List assigned case-files |
| API-021 | `GET` | `/case-files/{recordId}` | Get a case-file |
| API-022 | `PATCH` | `/case-files/{recordId}` | Update case-file data/status |
| API-023 | `GET` | `/case-files/{recordId}/timeline` | Get case-file timeline |
| API-028 | `POST` | `/case-files/{recordId}/disclosures` | Reveal a sensitive field |
| API-024 | `POST` | `/case-files/{recordId}/assignments` | Assign staff to case-file |
| API-025 | `POST` | `/case-files/{recordId}/assignments/{assignmentId}/revocations` | Revoke an assignment |
| API-026 | `GET` | `/case-files/{recordId}/departments/{departmentId}/notes` | List department notes |
| API-027 | `POST` | `/case-files/{recordId}/departments/{departmentId}/notes` | Create a note or correction |
| API-029 | `GET` | `/case-files/{recordId}/consents` | List consents |
| API-030 | `POST` | `/case-files/{recordId}/consents` | Record a consent |
| API-031 | `POST` | `/case-files/{recordId}/consents/{consentId}/revocations` | Revoke optional purposes |
| API-033 | `POST` | `/case-files/{recordId}/referrals` | Create an internal referral |

---

## 6. `caseaccessrequest/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-018 | `POST` | `/case-access-requests` | Request access to an existing case-file |
| API-019 | `GET` | `/case-access-requests` | List access requests |

---

## 7. `note/`

Offline sync only. Department notes by area belong to `casefile/`.

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-041 | `POST` | `/notes/sync` | Sync offline note drafts |

---

## 8. `referral/`

Inbox and transitions only. Referral creation belongs to `casefile/`.

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-032 | `GET` | `/referrals` | List referrals (inbox) |
| API-034 | `POST` | `/referrals/{referralId}/transitions` | Change referral status |

---

## 9. `auditlog/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-035 | `GET` | `/audit-logs` | Query activity log |

---

## 10. `auditalert/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-036 | `GET` | `/audit-alerts` | Query access alerts |

---

## 11. `analytics/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-037 | `GET` | `/analytics/summary` | Get aggregate indicators |
| API-038 | `GET` | `/analytics/heatmap` | Get sector heatmap |

---

## 12. `institutionalcontact/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-039 | `GET` | `/institutional-contacts` | Get institutional directory |

---

## 13. `helpevent/`

| API ID | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| API-040 | `POST` | `/help-events` | Record dialer open event |

---

## Summary

| Folder | # Endpoints |
|--------|:-----------:|
| `health` | 1 |
| `auth` | 5 |
| `user` | 8 |
| `catalog` | 1 |
| `casefile` | 15 |
| `caseaccessrequest` | 2 |
| `note` | 1 |
| `referral` | 2 |
| `auditlog` | 1 |
| `auditalert` | 1 |
| `analytics` | 2 |
| `institutionalcontact` | 1 |
| `helpevent` | 1 |
| **Total** | **41** |

Each folder exists under all 4 layers: `handlers/`, `middleware/`, `models/`, `services/`.
