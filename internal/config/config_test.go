package config

import "testing"

func TestConfigIsDev(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{env: "", want: false},
		{env: "production", want: false},
		{env: "prod", want: false},
		{env: "staging", want: false},
		{env: "unknown", want: false},
		{env: "dev", want: true},
		{env: "development", want: true},
		{env: "local", want: true},
		{env: " DEV ", want: true},
		{env: "Local", want: true},
		{env: "DEVELOPMENT", want: true},
	}

	for _, tc := range cases {
		cfg := Config{Env: tc.env}
		if got := cfg.IsDev(); got != tc.want {
			t.Errorf("Config{Env: %q}.IsDev() = %v, want %v", tc.env, got, tc.want)
		}
	}
}

func TestLoadAppEnvDefaultAndDev(t *testing.T) {
	t.Setenv("MFA_ENCRYPTION_KEY", "test-mfa-key")
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	t.Setenv("APP_ENV", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.IsDev() {
		t.Fatalf("expected IsDev()=false when APP_ENV is unset, got true (Env=%q)", cfg.Env)
	}
	if cfg.Env != "production" {
		t.Fatalf("expected default Env=%q, got %q", "production", cfg.Env)
	}

	t.Setenv("APP_ENV", "dev")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if !cfg.IsDev() {
		t.Fatalf("expected IsDev()=true when APP_ENV=dev, got false")
	}
}
