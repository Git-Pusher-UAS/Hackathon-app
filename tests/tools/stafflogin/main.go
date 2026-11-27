// Command stafflogin inicia sesión como personal en una API de prueba y
// escribe el JWT de la cookie sai_staff en stdout. Hace el flujo completo de
// §2 del contrato: login (202) -> enrolamiento TOTP -> verificación.
//
// Solo sirve para cuentas sin segundo factor (base recién sembrada en CI),
// porque calcula el código TOTP con el secreto que devuelve el enrolamiento.
//
//	go run ./tests/tools/stafflogin -base http://localhost:3000 -user admin -password ...
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
)

func post(base, path string, body any, out any) (int, []*http.Cookie, error) {
	raw, _ := json.Marshal(body)
	resp, err := http.Post(base+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return resp.StatusCode, nil, fmt.Errorf("%s: HTTP %d %s", path, resp.StatusCode, data)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, nil, err
		}
	}
	return resp.StatusCode, resp.Cookies(), nil
}

func run(base, user, password string) (string, error) {
	var ch struct {
		ChallengeID string `json:"challengeId"`
		Next        string `json:"next"`
	}
	if _, _, err := post(base, "/api/v1/auth/login", map[string]string{"usuario": user, "password": password}, &ch); err != nil {
		return "", err
	}
	if ch.Next != "enroll" {
		return "", fmt.Errorf("la cuenta ya tiene segundo factor (next=%s); esta herramienta solo enrola cuentas nuevas", ch.Next)
	}
	var en struct {
		EnrollmentID string `json:"enrollmentId"`
		OtpauthURI   string `json:"otpauthUri"`
	}
	if _, _, err := post(base, "/api/v1/auth/mfa/enrollments", map[string]string{"challengeId": ch.ChallengeID}, &en); err != nil {
		return "", err
	}
	u, err := url.Parse(en.OtpauthURI)
	if err != nil {
		return "", err
	}
	code, err := totp.GenerateCode(u.Query().Get("secret"), time.Now())
	if err != nil {
		return "", err
	}
	_, cookies, err := post(base, "/api/v1/auth/mfa/verify",
		map[string]string{"challengeId": ch.ChallengeID, "code": code, "enrollmentId": en.EnrollmentID}, nil)
	if err != nil {
		return "", err
	}
	for _, c := range cookies {
		if c.Name == "sai_staff" && c.Value != "" {
			return c.Value, nil
		}
	}
	return "", fmt.Errorf("la verificación no puso la cookie sai_staff")
}

func main() {
	base := flag.String("base", "http://localhost:3000", "URL base de la API")
	user := flag.String("user", os.Getenv("INITIAL_ADMIN_USERNAME"), "usuario")
	password := flag.String("password", os.Getenv("INITIAL_ADMIN_PASSWORD"), "contraseña")
	flag.Parse()
	token, err := run(*base, *user, *password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stafflogin:", err)
		os.Exit(1)
	}
	fmt.Println(token)
}
