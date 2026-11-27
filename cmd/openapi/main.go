// Command openapi genera el contrato api/openapi.yaml (OpenAPI 3) a partir de
// docs/swagger.json (Swagger 2, generado por swag desde las anotaciones).
//
//	swag init -g cmd/main.go -d . --parseDependency --parseInternal
//	go run ./cmd/openapi            # escribe api/openapi.yaml
//	go run ./cmd/openapi -check     # falla si api/openapi.yaml está desactualizado
//
// Además de convertir, renombra los esquemas a nombres legibles para los
// tipos generados del front (backend-hackathon_internal_models_casefile.Summary
// pasa a CasefileSummary) y fija el servidor relativo /api.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"
)

const modulePrefix = "backend-hackathon_internal_"

// Nombre legible de cada paquete de DTOs. Los vacíos no llevan prefijo.
var pkgNames = map[string]string{
	"models":                      "",
	"handlers":                    "",
	"httpx":                       "",
	"auth":                        "Auth",
	"models_analytics":            "Analytics",
	"models_auditlog":             "AuditLog",
	"models_caseaccessrequest":    "CaseAccessRequest",
	"models_casefile":             "Casefile",
	"models_catalog":              "Catalog",
	"models_helpevent":            "HelpEvent",
	"models_institutionalcontact": "InstitutionalContact",
	"models_note":                 "Note",
	"models_referral":             "Referral",
	"models_user":                 "User",
}

func schemaName(old string) (string, error) {
	name := strings.TrimPrefix(strings.ReplaceAll(old, modulePrefix, ""), "internal_")
	pkg, typ, ok := strings.Cut(name, ".")
	if !ok {
		return "", fmt.Errorf("esquema sin paquete: %s", old)
	}
	prefix, known := pkgNames[pkg]
	if !known {
		return "", fmt.Errorf("paquete sin nombre en pkgNames: %s (%s)", pkg, old)
	}
	// Genéricos: Page-models_casefile_NoteView -> PageNoteView.
	if base, arg, generic := strings.Cut(typ, "-"); generic {
		typ = base + arg[strings.LastIndex(arg, "_")+1:]
	}
	return prefix + typ, nil
}

func main() {
	in := flag.String("in", "docs/swagger.json", "Swagger 2 generado por swag")
	out := flag.String("out", "api/openapi.yaml", "contrato OpenAPI 3")
	check := flag.Bool("check", false, "no escribe; falla si -out difiere")
	flag.Parse()

	raw, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalf("leer %s: %v", *in, err)
	}
	raw, err = renameSchemas(raw)
	if err != nil {
		log.Fatal(err)
	}

	var v2 openapi2.T
	if err := json.Unmarshal(raw, &v2); err != nil {
		log.Fatalf("swagger 2: %v", err)
	}
	v3, err := openapi2conv.ToV3(&v2)
	if err != nil {
		log.Fatalf("convertir a OpenAPI 3: %v", err)
	}
	v3.Servers = openapi3.Servers{{URL: "/api", Description: "Base path único del contrato"}}
	// Swagger 2 solo admite apiKey; en OpenAPI 3 el JWT se declara como bearer.
	if v3.Components != nil && v3.Components.SecuritySchemes["BearerAuth"] != nil {
		v3.Components.SecuritySchemes["BearerAuth"] = &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme().
			WithDescription("JWT de POST /auth/login. Las escrituras exigen además X-CSRF-Token (csrf_token del login o GET /auth/csrf-token).")}
	}
	if err := v3.Validate(context.Background()); err != nil {
		log.Fatalf("contrato inválido: %v", err)
	}

	js, err := v3.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	var tree any
	if err := json.Unmarshal(js, &tree); err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	buf.WriteString("# GENERADO: no editar a mano. Fuente: anotaciones swag -> docs/swagger.json -> go run ./cmd/openapi\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(tree); err != nil {
		log.Fatal(err)
	}
	_ = enc.Close()

	if *check {
		cur, err := os.ReadFile(*out)
		if err != nil {
			log.Fatalf("leer %s: %v", *out, err)
		}
		if !bytes.Equal(bytes.ReplaceAll(cur, []byte("\r\n"), []byte("\n")), buf.Bytes()) {
			log.Fatalf("%s está desactualizado: corre swag init y go run ./cmd/openapi", *out)
		}
		fmt.Println("OK:", *out, "coincide con", *in)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("escrito", *out)
}

// renameSchemas cambia las claves de definitions y sus $ref.
func renameSchemas(raw []byte) ([]byte, error) {
	var doc struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	olds := make([]string, 0, len(doc.Definitions))
	for k := range doc.Definitions {
		olds = append(olds, k)
	}
	// Los más largos primero para que un nombre no sea prefijo de otro al reemplazar.
	sort.Slice(olds, func(i, j int) bool { return len(olds[i]) > len(olds[j]) })
	seen := map[string]string{}
	pairs := make([]string, 0, 4*len(olds))
	for _, old := range olds {
		name, err := schemaName(old)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[name]; dup {
			return nil, fmt.Errorf("colisión de nombres: %s y %s -> %s", prev, old, name)
		}
		seen[name] = old
		pairs = append(pairs, `"#/definitions/`+old+`"`, `"#/definitions/`+name+`"`, `"`+old+`":`, `"`+name+`":`)
	}
	return []byte(strings.NewReplacer(pairs...).Replace(string(raw))), nil
}
