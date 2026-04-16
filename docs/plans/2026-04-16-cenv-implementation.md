# cenv Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `cenv`, a Go CLI that manages isolated Claude Code configuration directories.

**Architecture:** Cobra CLI with internal packages for env operations, settings manipulation, auth detection, and bootstrap. Each package is independently testable. The CLI layer is thin glue.

**Tech Stack:** Go 1.26, cobra for CLI, standard library for JSON/file I/O/exec. No other dependencies.

**Spec:** `~/pickleton/projects/cenv/2026-04-15-cenv-design.md`

---

## File Structure

```
go.mod
go.sum
cmd/cenv/main.go                    CLI entrypoint, root command
internal/env/env.go                  Environment CRUD (create, remove, exists, path, list)
internal/env/env_test.go             Tests for env operations
internal/settings/settings.go        Settings show, get, merge
internal/settings/merge.go           Deep merge logic (isolated for testing)
internal/settings/merge_test.go      Deep merge tests
internal/settings/dotpath.go         Dot-path key extraction
internal/settings/dotpath_test.go    Dot-path tests
internal/settings/detect.go          JSON vs file path detection
internal/settings/detect_test.go     Detection tests
internal/settings/settings_test.go   Integration tests for show/get/merge
internal/auth/auth.go                Auth detection and auth env management
internal/auth/auth_test.go           Auth detection tests
internal/bootstrap/bootstrap.go      Onboarding config, settings seeding
internal/bootstrap/bootstrap_test.go Bootstrap tests
```

---

### Task 1: Project scaffolding and module init

**Files:**
- Create: `go.mod`
- Create: `cmd/cenv/main.go`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/technicalpickles/cenv`

- [ ] **Step 2: Create .gitignore**

```gitignore
# Binary
cenv
dist/

# OS
.DS_Store
```

- [ ] **Step 3: Install cobra**

Run: `go get github.com/spf13/cobra@latest`

- [ ] **Step 4: Create main.go with root command**

Create `cmd/cenv/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verify it builds**

Run: `go build ./cmd/cenv`
Expected: binary `cenv` created, `./cenv --help` prints usage

- [ ] **Step 6: Commit**

```
feat: project scaffolding with cobra root command
```

---

### Task 2: Environment package (core CRUD)

**Files:**
- Create: `internal/env/env.go`
- Create: `internal/env/env_test.go`

- [ ] **Step 1: Write failing tests for env operations**

Create `internal/env/env_test.go`:

```go
package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBasePath(t *testing.T) {
	t.Run("uses CENV_BASE when set", func(t *testing.T) {
		t.Setenv("CENV_BASE", "/tmp/custom-cenv")
		if got := BasePath(); got != "/tmp/custom-cenv" {
			t.Errorf("BasePath() = %q, want %q", got, "/tmp/custom-cenv")
		}
	})

	t.Run("defaults to ~/.local/share/cenv", func(t *testing.T) {
		t.Setenv("CENV_BASE", "")
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "share", "cenv")
		if got := BasePath(); got != want {
			t.Errorf("BasePath() = %q, want %q", got, want)
		}
	})
}

func TestEnvPath(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())
	got := Path("myenv")
	want := filepath.Join(BasePath(), "myenv")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestExists(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	if Exists("nope") {
		t.Error("Exists() returned true for non-existent env")
	}

	os.MkdirAll(filepath.Join(base, "yep"), 0o755)
	if !Exists("yep") {
		t.Error("Exists() returned false for existing env")
	}
}

func TestList(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	t.Run("empty base returns empty list", func(t *testing.T) {
		envs, err := List()
		if err != nil {
			t.Fatal(err)
		}
		if len(envs) != 0 {
			t.Errorf("List() returned %d envs, want 0", len(envs))
		}
	})

	t.Run("lists directories only", func(t *testing.T) {
		os.MkdirAll(filepath.Join(base, "alpha"), 0o755)
		os.MkdirAll(filepath.Join(base, "beta"), 0o755)
		os.WriteFile(filepath.Join(base, "not-an-env.txt"), []byte("hi"), 0o644)

		envs, err := List()
		if err != nil {
			t.Fatal(err)
		}
		if len(envs) != 2 {
			t.Errorf("List() returned %d envs, want 2", len(envs))
		}
	})
}

func TestRemove(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	t.Run("removes existing env", func(t *testing.T) {
		envDir := filepath.Join(base, "doomed")
		os.MkdirAll(envDir, 0o755)
		os.WriteFile(filepath.Join(envDir, "settings.json"), []byte("{}"), 0o644)

		if err := Remove("doomed"); err != nil {
			t.Fatal(err)
		}
		if Exists("doomed") {
			t.Error("env still exists after Remove()")
		}
	})

	t.Run("error on non-existent env", func(t *testing.T) {
		err := Remove("ghost")
		if err == nil {
			t.Error("Remove() should fail for non-existent env")
		}
	})
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"myenv", true},
		{"my-env", true},
		{"my_env", true},
		{"auth-anthropic", true},
		{"test123", true},
		{"", false},
		{"has spaces", false},
		{"has/slash", false},
		{".dotfile", false},
		{"../escape", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if tt.valid && err != nil {
				t.Errorf("ValidateName(%q) returned error: %v", tt.name, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateName(%q) should have returned error", tt.name)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/env/`
Expected: compilation error, package not found

- [ ] **Step 3: Implement env package**

Create `internal/env/env.go`:

```go
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// BasePath returns the root directory for all environments.
// Defaults to ~/.local/share/cenv, overridable via CENV_BASE.
func BasePath() string {
	if base := os.Getenv("CENV_BASE"); base != "" {
		return base
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cenv")
}

// Path returns the directory path for a named environment.
func Path(name string) string {
	return filepath.Join(BasePath(), name)
}

// Exists checks whether an environment directory exists.
func Exists(name string) bool {
	info, err := os.Stat(Path(name))
	return err == nil && info.IsDir()
}

// List returns the names of all environments in BasePath.
func List() ([]string, error) {
	base := BasePath()
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", base, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Remove deletes an environment directory.
func Remove(name string) error {
	if !Exists(name) {
		return fmt.Errorf("environment %q does not exist", name)
	}
	return os.RemoveAll(Path(name))
}

// ValidateName checks that a name is safe for use as a directory name.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid environment name %q: use letters, numbers, hyphens, underscores (must start with a letter)", name)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/env/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```
feat(env): core environment CRUD operations
```

---

### Task 3: Deep merge

**Files:**
- Create: `internal/settings/merge.go`
- Create: `internal/settings/merge_test.go`

- [ ] **Step 1: Write failing tests for deep merge**

Create `internal/settings/merge_test.go`:

```go
package settings

import (
	"encoding/json"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		overlay string
		want    string
	}{
		{
			name:    "empty base",
			base:    `{}`,
			overlay: `{"a": 1}`,
			want:    `{"a":1}`,
		},
		{
			name:    "empty overlay",
			base:    `{"a": 1}`,
			overlay: `{}`,
			want:    `{"a":1}`,
		},
		{
			name:    "scalar override",
			base:    `{"a": 1}`,
			overlay: `{"a": 2}`,
			want:    `{"a":2}`,
		},
		{
			name:    "nested merge",
			base:    `{"sandbox": {"enabled": true}}`,
			overlay: `{"sandbox": {"allowWrite": ["/tmp"]}}`,
			want:    `{"sandbox":{"allowWrite":["/tmp"],"enabled":true}}`,
		},
		{
			name:    "array replace not merge",
			base:    `{"tags": ["a", "b"]}`,
			overlay: `{"tags": ["c"]}`,
			want:    `{"tags":["c"]}`,
		},
		{
			name:    "new keys added",
			base:    `{"a": 1}`,
			overlay: `{"b": 2}`,
			want:    `{"a":1,"b":2}`,
		},
		{
			name:    "deeply nested",
			base:    `{"a": {"b": {"c": 1, "d": 2}}}`,
			overlay: `{"a": {"b": {"c": 3, "e": 4}}}`,
			want:    `{"a":{"b":{"c":3,"d":2,"e":4}}}`,
		},
		{
			name:    "overlay scalar replaces object",
			base:    `{"a": {"b": 1}}`,
			overlay: `{"a": "string"}`,
			want:    `{"a":"string"}`,
		},
		{
			name:    "overlay object replaces scalar",
			base:    `{"a": "string"}`,
			overlay: `{"a": {"b": 1}}`,
			want:    `{"a":{"b":1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base, overlay map[string]any
			json.Unmarshal([]byte(tt.base), &base)
			json.Unmarshal([]byte(tt.overlay), &overlay)

			result := DeepMerge(base, overlay)
			got, _ := json.Marshal(result)

			if string(got) != tt.want {
				t.Errorf("DeepMerge() = %s, want %s", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/settings/`
Expected: compilation error

- [ ] **Step 3: Implement DeepMerge**

Create `internal/settings/merge.go`:

```go
package settings

// DeepMerge recursively merges overlay into base.
// Objects are merged recursively. All other types (scalars, arrays)
// in overlay replace the corresponding value in base.
func DeepMerge(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := baseVal.(map[string]any)
			overlayMap, overlayIsMap := v.(map[string]any)
			if baseIsMap && overlayIsMap {
				result[k] = DeepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/settings/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```
feat(settings): deep merge for JSON objects
```

---

### Task 4: Dot-path key extraction

**Files:**
- Create: `internal/settings/dotpath.go`
- Create: `internal/settings/dotpath_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/settings/dotpath_test.go`:

```go
package settings

import (
	"encoding/json"
	"testing"
)

func TestGetByDotPath(t *testing.T) {
	data := `{
		"sandbox": {
			"enabled": true,
			"allowWrite": ["/tmp", "/var"]
		},
		"permissions": {
			"allow": ["Read", "Write"]
		},
		"topLevel": "hello"
	}`
	var obj map[string]any
	json.Unmarshal([]byte(data), &obj)

	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"topLevel", `"hello"`, false},
		{"sandbox.enabled", "true", false},
		{"sandbox.allowWrite", `["/tmp","/var"]`, false},
		{"sandbox", `{"allowWrite":["/tmp","/var"],"enabled":true}`, false},
		{"permissions.allow", `["Read","Write"]`, false},
		{"nonexistent", "", true},
		{"sandbox.nonexistent", "", true},
		{"sandbox.enabled.tooDeep", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := GetByDotPath(obj, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetByDotPath(%q) should have returned error", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByDotPath(%q) error: %v", tt.path, err)
			}
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != tt.want {
				t.Errorf("GetByDotPath(%q) = %s, want %s", tt.path, gotJSON, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/settings/`
Expected: compilation error

- [ ] **Step 3: Implement GetByDotPath**

Create `internal/settings/dotpath.go`:

```go
package settings

import (
	"fmt"
	"strings"
)

// GetByDotPath traverses a nested map using a dot-separated key path.
// Returns the value at the path, or an error if the path doesn't exist.
func GetByDotPath(data map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("key %q: cannot traverse into non-object", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", path)
		}
		current = val
	}
	return current, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/settings/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```
feat(settings): dot-path key extraction
```

---

### Task 5: JSON vs file path detection and settings file operations

**Files:**
- Create: `internal/settings/detect.go`
- Create: `internal/settings/detect_test.go`
- Create: `internal/settings/settings.go`
- Create: `internal/settings/settings_test.go`

- [ ] **Step 1: Write failing tests for detection**

Create `internal/settings/detect_test.go`:

```go
package settings

import "testing"

func TestIsJSON(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"key": "value"}`, true},
		{`  {"key": "value"}`, true},
		{"\t{}", true},
		{`[1, 2, 3]`, true},
		{`  [1]`, true},
		{"config.json", false},
		{"/path/to/file.json", false},
		{"", false},
		{"not json at all", false},
		{"true", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsJSON(tt.input); got != tt.want {
				t.Errorf("IsJSON(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Write failing tests for settings file operations**

Create `internal/settings/settings_test.go`:

```go
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"sandbox":{"enabled":true}}`), 0o644)

	data, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	sandbox, ok := data["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("sandbox key missing or wrong type")
	}
	if sandbox["enabled"] != true {
		t.Error("sandbox.enabled should be true")
	}
}

func TestLoad_missing(t *testing.T) {
	_, err := Load("/nonexistent/settings.json")
	if err == nil {
		t.Error("Load should fail for missing file")
	}
}

func TestLoad_invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`not json`), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Error("Load should fail for invalid JSON")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	data := map[string]any{"key": "value"}
	if err := Save(path, data); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	var loaded map[string]any
	json.Unmarshal(raw, &loaded)
	if loaded["key"] != "value" {
		t.Error("saved data doesn't match")
	}
}

func TestMergeInto(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"existing":"keep","nested":{"a":1}}`), 0o644)

	overlay := map[string]any{
		"new":    "added",
		"nested": map[string]any{"b": float64(2)},
	}
	if err := MergeInto(path, overlay); err != nil {
		t.Fatal(err)
	}

	data, _ := Load(path)
	if data["existing"] != "keep" {
		t.Error("existing key should be preserved")
	}
	if data["new"] != "added" {
		t.Error("new key should be added")
	}
	nested := data["nested"].(map[string]any)
	if nested["a"] != float64(1) {
		t.Error("nested.a should be preserved")
	}
	if nested["b"] != float64(2) {
		t.Error("nested.b should be added")
	}
}

func TestResolveOverlay_json(t *testing.T) {
	data, err := ResolveOverlay(`{"key":"value"}`)
	if err != nil {
		t.Fatal(err)
	}
	if data["key"] != "value" {
		t.Error("should parse inline JSON")
	}
}

func TestResolveOverlay_file(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.json")
	os.WriteFile(path, []byte(`{"from":"file"}`), 0o644)

	data, err := ResolveOverlay(path)
	if err != nil {
		t.Fatal(err)
	}
	if data["from"] != "file" {
		t.Error("should read from file")
	}
}

func TestResolveOverlay_badInput(t *testing.T) {
	_, err := ResolveOverlay("not-a-file-and-not-json")
	if err == nil {
		t.Error("should fail for input that's neither JSON nor valid file")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/settings/`
Expected: compilation errors

- [ ] **Step 4: Implement detect.go**

Create `internal/settings/detect.go`:

```go
package settings

import "strings"

// IsJSON returns true if the input looks like inline JSON
// (starts with { or [ after trimming whitespace).
func IsJSON(input string) bool {
	trimmed := strings.TrimLeft(input, " \t\n\r")
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
```

- [ ] **Step 5: Implement settings.go**

Create `internal/settings/settings.go`:

```go
package settings

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses a settings.json file.
func Load(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}
	return data, nil
}

// Save writes a map as pretty-printed JSON to a file.
func Save(path string, data map[string]any) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// MergeInto loads a settings file, deep merges overlay into it, and saves.
func MergeInto(path string, overlay map[string]any) error {
	base, err := Load(path)
	if err != nil {
		return err
	}
	merged := DeepMerge(base, overlay)
	return Save(path, merged)
}

// ResolveOverlay takes an argument that's either inline JSON or a file
// path, and returns the parsed map.
func ResolveOverlay(arg string) (map[string]any, error) {
	if IsJSON(arg) {
		var data map[string]any
		if err := json.Unmarshal([]byte(arg), &data); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return data, nil
	}

	if _, err := os.Stat(arg); err != nil {
		return nil, fmt.Errorf("argument is neither JSON nor a valid file path: %s", arg)
	}
	return Load(arg)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/settings/ -v`
Expected: all tests pass

- [ ] **Step 7: Commit**

```
feat(settings): load, save, merge, dot-path, and JSON detection
```

---

### Task 6: Bootstrap package

**Files:**
- Create: `internal/bootstrap/bootstrap.go`
- Create: `internal/bootstrap/bootstrap_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/bootstrap/bootstrap_test.go`:

```go
package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOnboarding(t *testing.T) {
	dir := t.TempDir()
	if err := WriteOnboarding(dir); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	json.Unmarshal(raw, &data)

	if data["hasCompletedOnboarding"] != true {
		t.Error("hasCompletedOnboarding should be true")
	}
}

func TestExtractAuth(t *testing.T) {
	t.Run("extracts awsAuthRefresh", func(t *testing.T) {
		settings := map[string]any{
			"awsAuthRefresh": map[string]any{"region": "us-west-2"},
			"permissions":    map[string]any{"allow": []any{"Read"}},
			"statusLine":     "custom",
		}
		auth := ExtractAuth(settings)
		if _, ok := auth["awsAuthRefresh"]; !ok {
			t.Error("should extract awsAuthRefresh")
		}
		if _, ok := auth["statusLine"]; !ok {
			t.Error("should extract statusLine")
		}
		if _, ok := auth["permissions"]; ok {
			t.Error("should not extract permissions")
		}
	})

	t.Run("extracts env vars", func(t *testing.T) {
		settings := map[string]any{
			"env": map[string]any{"AWS_PROFILE": "myprofile"},
		}
		auth := ExtractAuth(settings)
		if _, ok := auth["env"]; !ok {
			t.Error("should extract env")
		}
	})

	t.Run("empty when no auth settings", func(t *testing.T) {
		settings := map[string]any{
			"permissions": map[string]any{"allow": []any{"Read"}},
		}
		auth := ExtractAuth(settings)
		if len(auth) != 0 {
			t.Errorf("should be empty, got %v", auth)
		}
	})
}

func TestWriteSettings(t *testing.T) {
	dir := t.TempDir()
	data := map[string]any{"key": "value"}
	if err := WriteSettings(dir, data); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var loaded map[string]any
	json.Unmarshal(raw, &loaded)
	if loaded["key"] != "value" {
		t.Error("settings should match")
	}
}

func TestWriteEmptySettings(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSettings(dir, map[string]any{}); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var loaded map[string]any
	json.Unmarshal(raw, &loaded)
	if len(loaded) != 0 {
		t.Error("settings should be empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/bootstrap/`
Expected: compilation error

- [ ] **Step 3: Implement bootstrap package**

Create `internal/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// authKeys are the settings keys that carry authentication information.
var authKeys = []string{"env", "awsAuthRefresh", "statusLine"}

// WriteOnboarding writes a .claude.json that skips onboarding prompts.
func WriteOnboarding(envDir string) error {
	data := map[string]any{
		"hasCompletedOnboarding": true,
		"hasSeenTasksHint":      true,
		"theme":                 "dark",
		"numStartups":           float64(0),
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(filepath.Join(envDir, ".claude.json"), raw, 0o644)
}

// ExtractAuth returns only the auth-related keys from a settings map.
func ExtractAuth(settings map[string]any) map[string]any {
	auth := make(map[string]any)
	for _, key := range authKeys {
		if val, ok := settings[key]; ok {
			auth[key] = val
		}
	}
	return auth
}

// WriteSettings writes a settings.json into an environment directory.
func WriteSettings(envDir string, data map[string]any) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(filepath.Join(envDir, "settings.json"), raw, 0o644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bootstrap/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```
feat(bootstrap): onboarding config and auth extraction
```

---

### Task 7: Auth detection package

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/auth/auth_test.go`:

```go
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Run("detects AWS Bedrock", func(t *testing.T) {
		dir := t.TempDir()
		settings := map[string]any{
			"awsAuthRefresh": map[string]any{
				"region": "us-west-2",
			},
		}
		writeJSON(t, filepath.Join(dir, "settings.json"), settings)

		result, err := Detect(dir)
		if err != nil {
			t.Fatal(err)
		}
		if result.Type != "aws-bedrock" {
			t.Errorf("Type = %q, want %q", result.Type, "aws-bedrock")
		}
		if result.EnvName != "auth-aws-bedrock" {
			t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-aws-bedrock")
		}
	})

	t.Run("detects Anthropic account", func(t *testing.T) {
		dir := t.TempDir()
		writeJSON(t, filepath.Join(dir, "settings.json"), map[string]any{})
		claudeJSON := map[string]any{
			"oauthAccount": "josh.nichols@gusto.com",
		}
		writeJSON(t, filepath.Join(dir, ".claude.json"), claudeJSON)

		result, err := Detect(dir)
		if err != nil {
			t.Fatal(err)
		}
		if result.Type != "anthropic" {
			t.Errorf("Type = %q, want %q", result.Type, "anthropic")
		}
		if result.EnvName != "auth-anthropic" {
			t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-anthropic")
		}
	})

	t.Run("error when no auth detected", func(t *testing.T) {
		dir := t.TempDir()
		writeJSON(t, filepath.Join(dir, "settings.json"), map[string]any{})
		writeJSON(t, filepath.Join(dir, ".claude.json"), map[string]any{})

		_, err := Detect(dir)
		if err == nil {
			t.Error("should return error when no auth found")
		}
	})
}

func writeJSON(t *testing.T, path string, data map[string]any) {
	t.Helper()
	raw, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/auth/`
Expected: compilation error

- [ ] **Step 3: Implement auth package**

Create `internal/auth/auth.go`:

```go
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DetectResult holds the detected auth type and suggested environment name.
type DetectResult struct {
	Type    string // "anthropic" or "aws-bedrock"
	EnvName string // "auth-anthropic" or "auth-aws-bedrock"
	Detail  string // human-readable detail (email, region, etc.)
}

// Detect examines a Claude config directory and determines the auth method.
// configDir is typically ~/.claude/.
func Detect(configDir string) (*DetectResult, error) {
	// Check settings.json for AWS Bedrock
	settingsPath := filepath.Join(configDir, "settings.json")
	if raw, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]any
		if json.Unmarshal(raw, &settings) == nil {
			if aws, ok := settings["awsAuthRefresh"].(map[string]any); ok {
				region, _ := aws["region"].(string)
				detail := "AWS Bedrock"
				if region != "" {
					detail = fmt.Sprintf("AWS Bedrock (%s)", region)
				}
				return &DetectResult{
					Type:    "aws-bedrock",
					EnvName: "auth-aws-bedrock",
					Detail:  detail,
				}, nil
			}
		}
	}

	// Check .claude.json for Anthropic OAuth
	claudePath := filepath.Join(configDir, ".claude.json")
	if raw, err := os.ReadFile(claudePath); err == nil {
		var claude map[string]any
		if json.Unmarshal(raw, &claude) == nil {
			if account, ok := claude["oauthAccount"].(string); ok && account != "" {
				return &DetectResult{
					Type:    "anthropic",
					EnvName: "auth-anthropic",
					Detail:  fmt.Sprintf("Anthropic account (%s)", account),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no auth method detected in %s", configDir)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```
feat(auth): detect Anthropic vs AWS Bedrock auth
```

---

### Task 8: CLI commands - create, list, remove, path

**Files:**
- Modify: `cmd/cenv/main.go`

- [ ] **Step 1: Add create command**

Add to `cmd/cenv/main.go`, below the root command:

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

var createBare bool
var createAuth string
var createFrom string

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := env.ValidateName(name); err != nil {
			return err
		}
		if env.Exists(name) {
			return fmt.Errorf("environment %q already exists", name)
		}

		envDir := env.Path(name)
		if err := os.MkdirAll(envDir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Determine initial settings
		var initialSettings map[string]any

		switch {
		case createBare:
			initialSettings = map[string]any{}

		case createAuth != "":
			authEnvName := "auth-" + createAuth
			if !env.Exists(authEnvName) {
				os.RemoveAll(envDir)
				return fmt.Errorf("auth environment %q does not exist (run: cenv auth create)", authEnvName)
			}
			authSettings, err := settings.Load(filepath.Join(env.Path(authEnvName), "settings.json"))
			if err != nil {
				os.RemoveAll(envDir)
				return fmt.Errorf("reading auth settings: %w", err)
			}
			initialSettings = authSettings

		case createFrom == "user":
			home, _ := os.UserHomeDir()
			userSettings, err := settings.Load(filepath.Join(home, ".claude", "settings.json"))
			if err != nil {
				os.RemoveAll(envDir)
				return fmt.Errorf("reading user settings: %w", err)
			}
			initialSettings = userSettings

		case createFrom != "":
			if !env.Exists(createFrom) {
				os.RemoveAll(envDir)
				return fmt.Errorf("source environment %q does not exist", createFrom)
			}
			sourceSettings, err := settings.Load(filepath.Join(env.Path(createFrom), "settings.json"))
			if err != nil {
				os.RemoveAll(envDir)
				return fmt.Errorf("reading source settings: %w", err)
			}
			initialSettings = sourceSettings

		default:
			// Auto-detect auth from user config
			home, _ := os.UserHomeDir()
			claudeDir := filepath.Join(home, ".claude")
			userSettingsPath := filepath.Join(claudeDir, "settings.json")
			if userSettings, err := settings.Load(userSettingsPath); err == nil {
				initialSettings = bootstrap.ExtractAuth(userSettings)
			} else {
				initialSettings = map[string]any{}
			}
		}

		if err := bootstrap.WriteSettings(envDir, initialSettings); err != nil {
			os.RemoveAll(envDir)
			return err
		}
		if err := bootstrap.WriteOnboarding(envDir); err != nil {
			os.RemoveAll(envDir)
			return err
		}

		fmt.Fprintf(os.Stderr, "[cenv] Created environment %q\n", name)
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVar(&createBare, "bare", false, "Create with empty settings")
	createCmd.Flags().StringVar(&createAuth, "auth", "", "Use auth from named auth environment")
	createCmd.Flags().StringVar(&createFrom, "from", "", "Clone settings from 'user' or another environment")
	rootCmd.AddCommand(createCmd)
}
```

- [ ] **Step 2: Add list command**

```go
var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		envs, err := env.List()
		if err != nil {
			return err
		}
		if len(envs) == 0 {
			fmt.Println("No environments yet.")
			fmt.Println("Create one: cenv create <name>")
			return nil
		}
		for _, name := range envs {
			fmt.Println(name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

- [ ] **Step 3: Add remove command**

```go
var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := env.Remove(name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "[cenv] Removed environment %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
```

- [ ] **Step 4: Add path command**

```go
var pathCmd = &cobra.Command{
	Use:   "path <name>",
	Short: "Print environment directory path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		fmt.Println(env.Path(name))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pathCmd)
}
```

- [ ] **Step 5: Verify it builds and basic commands work**

Run: `go build ./cmd/cenv`
Then: `CENV_BASE=/tmp/cenv-test ./cenv create test-basic --bare`
Then: `CENV_BASE=/tmp/cenv-test ./cenv list`
Then: `CENV_BASE=/tmp/cenv-test ./cenv path test-basic`
Then: `CENV_BASE=/tmp/cenv-test ./cenv remove test-basic`
Clean up: `rm -rf /tmp/cenv-test`

- [ ] **Step 6: Commit**

```
feat(cli): create, list, remove, and path commands
```

---

### Task 9: CLI commands - settings subcommands

**Files:**
- Modify: `cmd/cenv/main.go`

- [ ] **Step 1: Add settings command group**

```go
var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage environment settings",
}

var settingsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Dump settings.json",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		data, err := settings.Load(filepath.Join(env.Path(name), "settings.json"))
		if err != nil {
			return err
		}
		raw, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(raw))
		return nil
	},
}

var settingsGetCmd = &cobra.Command{
	Use:   "get <name> <key>",
	Short: "Get a value by dot-path (e.g. sandbox.enabled)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, key := args[0], args[1]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		data, err := settings.Load(filepath.Join(env.Path(name), "settings.json"))
		if err != nil {
			return err
		}
		val, err := settings.GetByDotPath(data, key)
		if err != nil {
			return err
		}
		raw, _ := json.MarshalIndent(val, "", "  ")
		fmt.Println(string(raw))
		return nil
	},
}

var settingsMergeCmd = &cobra.Command{
	Use:   "merge <name> <json|file>",
	Short: "Deep merge into settings.json",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, arg := args[0], args[1]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		overlay, err := settings.ResolveOverlay(arg)
		if err != nil {
			return err
		}
		settingsPath := filepath.Join(env.Path(name), "settings.json")
		if err := settings.MergeInto(settingsPath, overlay); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "[cenv] Merged settings into %q\n", name)
		return nil
	},
}

func init() {
	settingsCmd.AddCommand(settingsShowCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsCmd.AddCommand(settingsMergeCmd)
	rootCmd.AddCommand(settingsCmd)
}
```

- [ ] **Step 2: Verify settings commands work**

Run: `go build ./cmd/cenv`
Then:
```bash
CENV_BASE=/tmp/cenv-test ./cenv create test-settings --bare
CENV_BASE=/tmp/cenv-test ./cenv settings show test-settings
CENV_BASE=/tmp/cenv-test ./cenv settings merge test-settings '{"sandbox":{"enabled":true}}'
CENV_BASE=/tmp/cenv-test ./cenv settings show test-settings
CENV_BASE=/tmp/cenv-test ./cenv settings get test-settings sandbox.enabled
CENV_BASE=/tmp/cenv-test ./cenv remove test-settings
rm -rf /tmp/cenv-test
```

- [ ] **Step 3: Commit**

```
feat(cli): settings show, get, and merge commands
```

---

### Task 10: CLI commands - run

**Files:**
- Modify: `cmd/cenv/main.go`

- [ ] **Step 1: Add run command**

```go
var runCmd = &cobra.Command{
	Use:                "run <name> [-- claude-args...]",
	Short:              "Launch Claude in an environment",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}

		envDir := env.Path(name)

		// Preflight: verify settings.json exists and is valid
		settingsPath := filepath.Join(envDir, "settings.json")
		if _, err := settings.Load(settingsPath); err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}

		// Separate claude args (everything after --)
		var claudeArgs []string
		if len(args) > 1 {
			if args[1] != "--" {
				return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
			}
			claudeArgs = args[2:]
		}

		fmt.Fprintf(os.Stderr, "[cenv] Using %q (%s)\n", name, envDir)

		// Find claude binary
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		// Build environment with CLAUDE_CONFIG_DIR set
		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		// Exec replaces the current process
		execArgs := append([]string{"claude"}, claudeArgs...)
		return syscall.Exec(claudePath, execArgs, environ)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
```

Note: add `"os/exec"` and `"syscall"` to imports.

- [ ] **Step 2: Verify run command builds**

Run: `go build ./cmd/cenv`
Then test preflight error:
```bash
CENV_BASE=/tmp/cenv-test ./cenv create test-run --bare
CENV_BASE=/tmp/cenv-test ./cenv run test-run
# Should print: [cenv] Using "test-run" (...) and try to exec claude
CENV_BASE=/tmp/cenv-test ./cenv run nonexistent
# Should print: environment "nonexistent" does not exist
CENV_BASE=/tmp/cenv-test ./cenv remove test-run
rm -rf /tmp/cenv-test
```

- [ ] **Step 3: Commit**

```
feat(cli): run command with preflight checks and -- separator
```

---

### Task 11: CLI commands - auth subcommands

**Files:**
- Modify: `cmd/cenv/main.go`

- [ ] **Step 1: Add auth command group**

```go
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage auth environments",
}

var authCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Detect current auth and create an auth environment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		claudeDir := filepath.Join(home, ".claude")

		result, err := auth.Detect(claudeDir)
		if err != nil {
			return fmt.Errorf("could not detect auth: %w", err)
		}

		fmt.Fprintf(os.Stderr, "[cenv] Detected: %s\n", result.Detail)

		envDir := env.Path(result.EnvName)
		alreadyExists := env.Exists(result.EnvName)

		if err := os.MkdirAll(envDir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Read user settings and extract auth
		userSettings, err := settings.Load(filepath.Join(claudeDir, "settings.json"))
		if err != nil {
			return fmt.Errorf("reading user settings: %w", err)
		}
		authSettings := bootstrap.ExtractAuth(userSettings)

		if err := bootstrap.WriteSettings(envDir, authSettings); err != nil {
			return err
		}
		if err := bootstrap.WriteOnboarding(envDir); err != nil {
			return err
		}

		if alreadyExists {
			fmt.Fprintf(os.Stderr, "[cenv] Updated environment %q\n", result.EnvName)
		} else {
			fmt.Fprintf(os.Stderr, "[cenv] Created environment %q\n", result.EnvName)
		}
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List auth environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		envs, err := env.List()
		if err != nil {
			return err
		}
		found := false
		for _, name := range envs {
			if len(name) > 5 && name[:5] == "auth-" {
				fmt.Println(name)
				found = true
			}
		}
		if !found {
			fmt.Println("No auth environments yet.")
			fmt.Println("Create one: cenv auth create")
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authCreateCmd)
	authCmd.AddCommand(authListCmd)
	rootCmd.AddCommand(authCmd)
}
```

- [ ] **Step 2: Verify auth commands build**

Run: `go build ./cmd/cenv`
Then: `./cenv auth create` (should detect from your real ~/.claude/)
Then: `./cenv auth list`

- [ ] **Step 3: Commit**

```
feat(cli): auth create and list commands
```

---

### Task 12: Split main.go into separate command files

The single `main.go` is getting large. Split commands into separate files for maintainability.

**Files:**
- Create: `cmd/cenv/create.go`
- Create: `cmd/cenv/list.go`
- Create: `cmd/cenv/remove.go`
- Create: `cmd/cenv/path.go`
- Create: `cmd/cenv/run.go`
- Create: `cmd/cenv/settings.go`
- Create: `cmd/cenv/auth.go`
- Modify: `cmd/cenv/main.go` (strip to just root command and main func)

- [ ] **Step 1: Move each command to its own file**

Each file in `cmd/cenv/` is `package main` and registers itself via `init()`. Move the command definitions from main.go into the corresponding files. The root command and `main()` stay in `main.go`.

- [ ] **Step 2: Verify it still builds and all commands work**

Run: `go build ./cmd/cenv`
Then: `./cenv --help`
Expected: all subcommands listed

- [ ] **Step 3: Commit**

```
refactor(cli): split commands into separate files
```

---

### Task 13: Run all tests, verify everything works end-to-end

- [ ] **Step 1: Run the full test suite**

Run: `go test ./... -v`
Expected: all tests pass

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 3: End-to-end manual test**

```bash
CENV_BASE=/tmp/cenv-e2e ./cenv create test-e2e --bare
CENV_BASE=/tmp/cenv-e2e ./cenv settings merge test-e2e '{"sandbox":{"enabled":true},"permissions":{"allow":["Read"]}}'
CENV_BASE=/tmp/cenv-e2e ./cenv settings get test-e2e sandbox.enabled
CENV_BASE=/tmp/cenv-e2e ./cenv settings show test-e2e
CENV_BASE=/tmp/cenv-e2e ./cenv list
CENV_BASE=/tmp/cenv-e2e ./cenv path test-e2e
CENV_BASE=/tmp/cenv-e2e ./cenv remove test-e2e
rm -rf /tmp/cenv-e2e
```

- [ ] **Step 4: Commit any fixes, then tag v0.1.0**

```
chore: v0.1.0 - initial release
```
