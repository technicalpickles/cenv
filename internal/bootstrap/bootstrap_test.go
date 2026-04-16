package bootstrap_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/bootstrap"
)

func TestWriteOnboarding(t *testing.T) {
	dir := t.TempDir()

	if err := bootstrap.WriteOnboarding(dir); err != nil {
		t.Fatalf("WriteOnboarding returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".claude.json"))
	if err != nil {
		t.Fatalf("failed to read .claude.json: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to parse .claude.json: %v", err)
	}

	if got["hasCompletedOnboarding"] != true {
		t.Errorf("hasCompletedOnboarding = %v, want true", got["hasCompletedOnboarding"])
	}
	if got["hasSeenTasksHint"] != true {
		t.Errorf("hasSeenTasksHint = %v, want true", got["hasSeenTasksHint"])
	}
	if got["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", got["theme"])
	}
	if got["numStartups"] != float64(0) {
		t.Errorf("numStartups = %v, want 0", got["numStartups"])
	}
}

func TestExtractAuth_AwsAuthRefresh(t *testing.T) {
	input := map[string]any{
		"awsAuthRefresh": map[string]any{"cmd": "refresh"},
		"permissions":    map[string]any{"allow": []string{"read"}},
	}

	got := bootstrap.ExtractAuth(input)

	if _, ok := got["awsAuthRefresh"]; !ok {
		t.Error("expected awsAuthRefresh to be present")
	}
	if _, ok := got["permissions"]; ok {
		t.Error("expected permissions to be absent")
	}
}

func TestExtractAuth_Env(t *testing.T) {
	input := map[string]any{
		"env":         map[string]any{"AWS_REGION": "us-east-1"},
		"apiKeyHelper": "some-helper",
	}

	got := bootstrap.ExtractAuth(input)

	if _, ok := got["env"]; !ok {
		t.Error("expected env to be present")
	}
	if _, ok := got["apiKeyHelper"]; ok {
		t.Error("expected apiKeyHelper to be absent")
	}
}

func TestExtractAuth_StatusLine(t *testing.T) {
	input := map[string]any{
		"statusLine": "some status",
		"theme":      "dark",
	}

	got := bootstrap.ExtractAuth(input)

	if _, ok := got["statusLine"]; !ok {
		t.Error("expected statusLine to be present")
	}
	if _, ok := got["theme"]; ok {
		t.Error("expected theme to be absent")
	}
}

func TestExtractAuth_EmptyWhenNoAuthKeys(t *testing.T) {
	input := map[string]any{
		"permissions": map[string]any{"allow": []string{"read"}},
		"theme":       "dark",
	}

	got := bootstrap.ExtractAuth(input)

	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestExtractAuth_DoesNotExtractNonAuthKeys(t *testing.T) {
	input := map[string]any{
		"permissions": map[string]any{"allow": []string{"*"}},
		"env":         map[string]any{"KEY": "value"},
	}

	got := bootstrap.ExtractAuth(input)

	if _, ok := got["permissions"]; ok {
		t.Error("expected permissions to be absent")
	}
	if _, ok := got["env"]; !ok {
		t.Error("expected env to be present")
	}
}

func TestWriteSettings_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	data := map[string]any{
		"awsAuthRefresh": map[string]any{"cmd": "refresh"},
	}

	if err := bootstrap.WriteSettings(dir, data); err != nil {
		t.Fatalf("WriteSettings returned error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	if _, ok := got["awsAuthRefresh"]; !ok {
		t.Error("expected awsAuthRefresh in settings.json")
	}
}

func TestWriteSettings_EmptyMap(t *testing.T) {
	dir := t.TempDir()

	if err := bootstrap.WriteSettings(dir, map[string]any{}); err != nil {
		t.Fatalf("WriteSettings returned error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty object, got %v", got)
	}
}
