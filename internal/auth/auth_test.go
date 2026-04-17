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

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "aws-bedrock" {
		t.Errorf("Type = %q, want %q", result.Type, "aws-bedrock")
	}
	if result.EnvName != "auth-aws-bedrock" {
		t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-aws-bedrock")
	}
	if result.Detail == "" {
		t.Error("Detail is empty, want region info")
	}
	want := "us-west-2"
	if result.Detail != want && !contains(result.Detail, want) {
		t.Errorf("Detail = %q, want it to contain %q", result.Detail, want)
	}
}

func TestDetect_AWSBedrock_NoRegion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{
		"awsAuthRefresh": {}
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "aws-bedrock" {
		t.Errorf("Type = %q, want %q", result.Type, "aws-bedrock")
	}
	if result.EnvName != "auth-aws-bedrock" {
		t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-aws-bedrock")
	}
}

func TestDetect_Anthropic(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": "user@example.com"
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
	if result.EnvName != "auth-anthropic" {
		t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-anthropic")
	}
	want := "user@example.com"
	if result.Detail != want && !contains(result.Detail, want) {
		t.Errorf("Detail = %q, want it to contain %q", result.Detail, want)
	}
}

func TestDetect_NoAuth(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{}`)

	_, err := auth.Detect(dir)
	if err == nil {
		t.Error("expected error when no auth found, got nil")
	}
}

func TestDetect_MissingSettingsFile(t *testing.T) {
	dir := t.TempDir()
	// Only .claude.json with oauthAccount, no settings.json
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": "user@example.com"
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
}

func TestDetect_EmptyOAuthAccount(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": ""
	}`)

	_, err := auth.Detect(dir)
	if err == nil {
		t.Error("expected error for empty oauthAccount, got nil")
	}
}

func TestDetect_Anthropic_ObjectShape(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {
			"accountUuid": "abc-123",
			"emailAddress": "user@example.com",
			"organizationUuid": "org-456"
		}
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
	if result.EnvName != "auth-anthropic" {
		t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-anthropic")
	}
	want := "user@example.com"
	if result.Detail != want {
		t.Errorf("Detail = %q, want %q", result.Detail, want)
	}
}

func TestDetect_Anthropic_ObjectShape_NoEmail(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {
			"accountUuid": "abc-123"
		}
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
	if result.Detail != "" {
		t.Errorf("Detail = %q, want empty string when emailAddress absent", result.Detail)
	}
}

func TestDetect_Anthropic_NullOAuthAccount(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": null
	}`)

	_, err := auth.Detect(dir)
	if err == nil {
		t.Error("expected error for null oauthAccount, got nil")
	}
}

func TestDetect_Anthropic_ObjectShape_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {}
	}`)

	_, err := auth.Detect(dir)
	if err == nil {
		t.Error("expected error for empty oauthAccount object, got nil")
	}
}

// contains is a helper for substring checks.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
