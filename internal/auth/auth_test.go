package auth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/auth"
)

func writeJSON(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
}

func TestDetect_AWSBedrock(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{
		"awsAuthRefresh": {
			"region": "us-west-2"
		}
	}`)

	if err := auth.Detect(dir); err != nil {
		t.Errorf("Detect() = %v, want nil for Bedrock config", err)
	}
}

func TestDetect_AWSBedrock_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{"awsAuthRefresh": {}}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error for empty awsAuthRefresh object")
	}
}

func TestDetect_Anthropic_StringShape(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{"oauthAccount": "user@example.com"}`)

	if err := auth.Detect(dir); err != nil {
		t.Errorf("Detect() = %v, want nil for legacy string oauthAccount", err)
	}
}

func TestDetect_Anthropic_ObjectShape(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {
			"accountUuid": "abc-123",
			"emailAddress": "user@example.com"
		}
	}`)

	if err := auth.Detect(dir); err != nil {
		t.Errorf("Detect() = %v, want nil for object oauthAccount", err)
	}
}

func TestDetect_NoAuth(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error when no auth present")
	}
}

func TestDetect_MissingSettingsFile(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, ".claude.json", `{"oauthAccount": "user@example.com"}`)

	if err := auth.Detect(dir); err != nil {
		t.Errorf("Detect() = %v, want nil when oauth present and settings.json missing", err)
	}
}

func TestDetect_EmptyOAuthString(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{"oauthAccount": ""}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error for empty oauthAccount string")
	}
}

func TestDetect_EmptyOAuthObject(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{"oauthAccount": {}}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error for empty oauthAccount object")
	}
}

func TestDetect_NullOAuthAccount(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{"oauthAccount": null}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error for null oauthAccount")
	}
}

// A token or key exported in the dev shell would make the no-auth tests above fail.
func TestMain(m *testing.M) {
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Exit(m.Run())
}

func TestDetect_ProcessEnvToken(t *testing.T) {
	for _, name := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeJSON(t, dir, "settings.json", `{}`)
			writeJSON(t, dir, ".claude.json", `{}`)
			t.Setenv(name, "secret")

			if err := auth.Detect(dir); err != nil {
				t.Errorf("Detect() = %v, want nil when %s is set", err, name)
			}
		})
	}
}

func TestDetect_EmptyProcessEnvToken(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error when token env var is empty")
	}
}

func TestDetect_SettingsEnvToken(t *testing.T) {
	for _, name := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeJSON(t, dir, "settings.json", `{"env": {"`+name+`": "secret"}}`)

			if err := auth.Detect(dir); err != nil {
				t.Errorf("Detect() = %v, want nil when settings.json env sets %s", err, name)
			}
		})
	}
}

func TestDetect_EmptySettingsEnvToken(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{"env": {"ANTHROPIC_AUTH_TOKEN": ""}}`)

	if err := auth.Detect(dir); err == nil {
		t.Error("Detect() = nil, want error when settings.json env token is empty")
	}
}
