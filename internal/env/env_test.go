package env_test

import (
	"os"
	"path/filepath"
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

		if err := env.Remove("nonexistent"); err == nil {
			t.Error("Remove(\"nonexistent\") expected error, got nil")
		}
	})
}

func TestValidateName(t *testing.T) {
	valid := []string{
		"myenv",
		"my-env",
		"my_env",
		"auth-anthropic",
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
