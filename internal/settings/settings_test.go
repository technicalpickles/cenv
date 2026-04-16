package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(path, []byte(`{"key": "value", "num": 42}`), 0644); err != nil {
			t.Fatal(err)
		}

		data, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data["key"] != "value" {
			t.Errorf("got key=%v, want %q", data["key"], "value")
		}
		if data["num"] != float64(42) {
			t.Errorf("got num=%v, want 42", data["num"])
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := Load("/nonexistent/path/settings.json")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte(`not json`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load(path)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("writes pretty-printed JSON with trailing newline", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.json")
		data := map[string]any{"key": "value"}

		if err := Save(path, data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// Must end with newline
		if len(content) == 0 || content[len(content)-1] != '\n' {
			t.Errorf("file does not end with newline: %q", string(content))
		}

		// Must be valid JSON
		var parsed map[string]any
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Errorf("saved content is not valid JSON: %v", err)
		}

		// Must be pretty-printed (contains newlines inside)
		if string(content) == `{"key":"value"}`+"\n" {
			t.Error("expected pretty-printed JSON, got compact")
		}

		if parsed["key"] != "value" {
			t.Errorf("got key=%v, want %q", parsed["key"], "value")
		}
	})
}

func TestMergeInto(t *testing.T) {
	t.Run("merges overlay into existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		base := map[string]any{"a": "1", "nested": map[string]any{"x": "old"}}
		if err := Save(path, base); err != nil {
			t.Fatal(err)
		}

		overlay := map[string]any{"b": "2", "nested": map[string]any{"y": "new"}}
		if err := MergeInto(path, overlay); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if result["a"] != "1" {
			t.Errorf("base key a lost, got %v", result["a"])
		}
		if result["b"] != "2" {
			t.Errorf("overlay key b not merged, got %v", result["b"])
		}
		nested, ok := result["nested"].(map[string]any)
		if !ok {
			t.Fatal("nested is not a map")
		}
		if nested["x"] != "old" {
			t.Errorf("nested.x lost, got %v", nested["x"])
		}
		if nested["y"] != "new" {
			t.Errorf("nested.y not merged, got %v", nested["y"])
		}
	})

	t.Run("returns error if file missing", func(t *testing.T) {
		err := MergeInto("/nonexistent/settings.json", map[string]any{"k": "v"})
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestResolveOverlay(t *testing.T) {
	t.Run("parses inline JSON string", func(t *testing.T) {
		result, err := ResolveOverlay(`{"key": "value"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("got key=%v, want %q", result["key"], "value")
		}
	})

	t.Run("loads from file path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "overlay.json")
		if err := os.WriteFile(path, []byte(`{"from": "file"}`), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ResolveOverlay(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["from"] != "file" {
			t.Errorf("got from=%v, want %q", result["from"], "file")
		}
	})

	t.Run("returns error for non-existent file path", func(t *testing.T) {
		_, err := ResolveOverlay("/nonexistent/overlay.json")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("returns error for invalid inline JSON", func(t *testing.T) {
		_, err := ResolveOverlay(`{bad json}`)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}
