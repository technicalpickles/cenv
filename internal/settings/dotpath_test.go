package settings

import (
	"testing"
)

var testData = map[string]any{
	"sandbox": map[string]any{
		"enabled":    true,
		"allowWrite": []any{"/tmp", "/var"},
	},
	"permissions": map[string]any{
		"allow": []any{"Read", "Write"},
	},
	"topLevel": "hello",
}

func TestGetByDotPath_TopLevel(t *testing.T) {
	val, err := GetByDotPath(testData, "topLevel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Errorf("expected %q, got %v", "hello", val)
	}
}

func TestGetByDotPath_NestedScalar(t *testing.T) {
	val, err := GetByDotPath(testData, "sandbox.enabled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestGetByDotPath_NestedArray(t *testing.T) {
	val, err := GetByDotPath(testData, "sandbox.allowWrite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := val.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", val)
	}
	if len(arr) != 2 || arr[0] != "/tmp" || arr[1] != "/var" {
		t.Errorf("expected [\"/tmp\", \"/var\"], got %v", arr)
	}
}

func TestGetByDotPath_NestedObject(t *testing.T) {
	val, err := GetByDotPath(testData, "sandbox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}
	if obj["enabled"] != true {
		t.Errorf("expected sandbox.enabled = true, got %v", obj["enabled"])
	}
}

func TestGetByDotPath_TwoLevelNested(t *testing.T) {
	val, err := GetByDotPath(testData, "permissions.allow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := val.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", val)
	}
	if len(arr) != 2 || arr[0] != "Read" || arr[1] != "Write" {
		t.Errorf("expected [\"Read\", \"Write\"], got %v", arr)
	}
}

func TestGetByDotPath_NonExistentKey(t *testing.T) {
	_, err := GetByDotPath(testData, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key, got nil")
	}
}

func TestGetByDotPath_NonExistentNestedKey(t *testing.T) {
	_, err := GetByDotPath(testData, "sandbox.nonexistent")
	if err == nil {
		t.Error("expected error for non-existent nested key, got nil")
	}
}

func TestGetByDotPath_TooDeep(t *testing.T) {
	_, err := GetByDotPath(testData, "sandbox.enabled.tooDeep")
	if err == nil {
		t.Error("expected error when traversing into a scalar, got nil")
	}
}
