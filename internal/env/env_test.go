package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/technicalpickles/cenv/internal/env"
)

func TestBasePath(t *testing.T) {
	t.Run("uses CENV_BASE when set", func(t *testing.T) {
		t.Setenv("CENV_BASE", "/tmp/custom-cenv")
		got := env.BasePath()
		if got != "/tmp/custom-cenv" {
			t.Errorf("BasePath() = %q, want %q", got, "/tmp/custom-cenv")
		}
	})

	t.Run("defaults to ~/.local/share/cenv", func(t *testing.T) {
		t.Setenv("CENV_BASE", "")
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, ".local", "share", "cenv")
		got := env.BasePath()
		if got != want {
			t.Errorf("BasePath() = %q, want %q", got, want)
		}
	})
}

func TestPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CENV_BASE", tmp)

	got := env.Path("myenv")
	want := filepath.Join(tmp, "myenv")
	if got != want {
		t.Errorf("Path(%q) = %q, want %q", "myenv", got, want)
	}
}

func TestExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CENV_BASE", tmp)

	t.Run("returns false for missing env", func(t *testing.T) {
		if env.Exists("nonexistent") {
			t.Error("Exists(\"nonexistent\") = true, want false")
		}
	})

	t.Run("returns true for existing env", func(t *testing.T) {
		if err := os.Mkdir(filepath.Join(tmp, "myenv"), 0755); err != nil {
			t.Fatal(err)
		}
		if !env.Exists("myenv") {
			t.Error("Exists(\"myenv\") = false, want true")
		}
	})
}

func TestList(t *testing.T) {
	t.Run("returns empty for empty base", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		names, err := env.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(names) != 0 {
			t.Errorf("List() = %v, want empty slice", names)
		}
	})

	t.Run("lists only directories", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		// Create directories (envs)
		if err := os.Mkdir(filepath.Join(tmp, "alpha"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tmp, "beta"), 0755); err != nil {
			t.Fatal(err)
		}
		// Create a file that should be ignored
		if err := os.WriteFile(filepath.Join(tmp, "notanenv.txt"), []byte("hi"), 0644); err != nil {
			t.Fatal(err)
		}

		names, err := env.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(names) != 2 {
			t.Errorf("List() returned %d items, want 2: %v", len(names), names)
		}
		found := map[string]bool{}
		for _, n := range names {
			found[n] = true
		}
		if !found["alpha"] || !found["beta"] {
			t.Errorf("List() = %v, want [alpha beta]", names)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("removes existing env", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		if err := os.Mkdir(filepath.Join(tmp, "myenv"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := env.Remove("myenv"); err != nil {
			t.Errorf("Remove(%q) unexpected error: %v", "myenv", err)
		}
		if env.Exists("myenv") {
			t.Error("env still exists after Remove")
		}
	})

	t.Run("errors on non-existent env", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		err := env.Remove("nonexistent")
		if err == nil {
			t.Fatal("Remove(\"nonexistent\") expected error, got nil")
		}
		if !strings.Contains(err.Error(), "environment \"nonexistent\" not found") {
			t.Errorf("Remove() error = %q, want it to contain %q", err.Error(), "environment \"nonexistent\" not found")
		}
	})
}

func TestInspect(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CENV_BASE", tmp)

	t.Run("errors on missing env", func(t *testing.T) {
		if _, err := env.Inspect("nope"); err == nil {
			t.Error("Inspect(missing) expected error, got nil")
		}
	})

	t.Run("returns metadata for env", func(t *testing.T) {
		envDir := filepath.Join(tmp, "myenv")
		if err := os.Mkdir(envDir, 0755); err != nil {
			t.Fatal(err)
		}
		// settings.json with awsAuthRefresh → has_auth should be true
		settings := `{"awsAuthRefresh": {"region": "us-west-2"}}`
		if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settings), 0644); err != nil {
			t.Fatal(err)
		}

		info, err := env.Inspect("myenv")
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if info.Name != "myenv" {
			t.Errorf("Name = %q, want %q", info.Name, "myenv")
		}
		if info.Path != envDir {
			t.Errorf("Path = %q, want %q", info.Path, envDir)
		}
		if !info.HasAuth {
			t.Error("HasAuth = false, want true (awsAuthRefresh present)")
		}
		if info.Size != int64(len(settings)) {
			t.Errorf("Size = %d, want %d", info.Size, len(settings))
		}
		if info.Mtime.IsZero() {
			t.Error("Mtime is zero")
		}
	})

	t.Run("has_auth false for bare env", func(t *testing.T) {
		envDir := filepath.Join(tmp, "bare")
		if err := os.Mkdir(envDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}

		info, err := env.Inspect("bare")
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if info.HasAuth {
			t.Error("HasAuth = true, want false (empty settings, no oauthAccount)")
		}
	})
}

func TestValidateName(t *testing.T) {
	valid := []string{
		"myenv",
		"my-env",
		"my_env",
		"shared-env",
		"test123",
	}
	for _, name := range valid {
		t.Run("valid: "+name, func(t *testing.T) {
			if err := env.ValidateName(name); err != nil {
				t.Errorf("ValidateName(%q) unexpected error: %v", name, err)
			}
		})
	}

	invalid := []string{
		"",
		"my env",
		"my/env",
		".dotfile",
		"../traversal",
	}
	for _, name := range invalid {
		t.Run("invalid: "+name, func(t *testing.T) {
			if err := env.ValidateName(name); err == nil {
				t.Errorf("ValidateName(%q) expected error, got nil", name)
			}
		})
	}
}
