package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchemaName(t *testing.T) {
	for old, want := range map[string]string{
		"backend-hackathon_internal_models_casefile.Summary":                                                  "CasefileSummary",
		"backend-hackathon_internal_models_casefile.Page-backend-hackathon_internal_models_casefile_NoteView": "CasefilePageNoteView",
		"backend-hackathon_internal_models.EstadoAlerta":                                                      "EstadoAlerta",
		"backend-hackathon_internal_httpx.ErrorResponse":                                                      "ErrorResponse",
		"backend-hackathon_internal_auth.LoginResult":                                                         "AuthLoginResult",
		"internal_handlers.LoginRequest":                                                                      "LoginRequest",
		"backend-hackathon_internal_models_institutionalcontact.Response":                                     "InstitutionalContactResponse",
	} {
		got, err := schemaName(old)
		if err != nil || got != want {
			t.Errorf("schemaName(%q) = %q, %v; quiero %q", old, got, err, want)
		}
	}
	if _, err := schemaName("backend-hackathon_internal_models_nuevo.X"); err == nil {
		t.Error("un paquete sin nombre en pkgNames debía fallar")
	}
}

func TestRenameSchemasRewritesRefs(t *testing.T) {
	in := `{"definitions":{"backend-hackathon_internal_models_note.SyncNote":{"type":"object"},` +
		`"backend-hackathon_internal_models_note.SyncRequest":{"properties":{"notes":{"items":{"$ref":"#/definitions/backend-hackathon_internal_models_note.SyncNote"}}}}}}`
	out, err := renameSchemas([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "backend-hackathon") {
		t.Fatalf("quedaron nombres viejos: %s", out)
	}
	var doc struct {
		Definitions map[string]any `json:"definitions"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Definitions["NoteSyncRequest"]; !ok || !strings.Contains(string(out), `"#/definitions/NoteSyncNote"`) {
		t.Fatalf("renombrado incompleto: %s", out)
	}
}
