package config

import "testing"

func TestParseEnvironment(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  Environment
		ok    bool
		isPro bool
	}{
		{"empty defaults to development", "", EnvDevelopment, true, false},
		{"whitespace defaults to development", "   ", EnvDevelopment, true, false},
		{"dev alias", "dev", EnvDevelopment, true, false},
		{"develop alias", "develop", EnvDevelopment, true, false},
		{"development canonical", "development", EnvDevelopment, true, false},
		{"mixed case development", "Development", EnvDevelopment, true, false},
		{"test", "test", EnvTest, true, false},
		{"testing alias", "testing", EnvTest, true, false},
		{"prod alias normalizes", "prod", EnvProduction, true, true},
		{"production canonical", "production", EnvProduction, true, true},
		{"upper PROD", "PROD", EnvProduction, true, true},
		{"padded production", "  production  ", EnvProduction, true, true},
		{"unknown falls back to default but not ok", "staging", EnvDevelopment, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseEnvironment(tc.in)
			if got != tc.want {
				t.Fatalf("ParseEnvironment(%q) env = %q, want %q", tc.in, got, tc.want)
			}
			if ok != tc.ok {
				t.Fatalf("ParseEnvironment(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if got.IsProduction() != tc.isPro {
				t.Fatalf("%q IsProduction() = %v, want %v", got, got.IsProduction(), tc.isPro)
			}
		})
	}
}

func TestEnvironmentValidate(t *testing.T) {
	valid := []Environment{EnvDevelopment, EnvTest, EnvProduction}
	for _, e := range valid {
		if !e.Validate() {
			t.Fatalf("expected %q to be valid", e)
		}
	}
	invalid := []Environment{"", "staging", "prod", "PROD", "Production"}
	for _, e := range invalid {
		if e.Validate() {
			t.Fatalf("expected %q to be invalid (Validate operates on canonical values)", e)
		}
	}
}

func TestEnvironmentString(t *testing.T) {
	if EnvProduction.String() != "production" {
		t.Fatalf("unexpected String(): %q", EnvProduction.String())
	}
	if DefaultEnvironment != EnvDevelopment {
		t.Fatalf("DefaultEnvironment must be development, got %q", DefaultEnvironment)
	}
}
