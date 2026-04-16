package settings_test

import (
	"encoding/json"
	"testing"

	"github.com/technicalpickles/cenv/internal/settings"
)

func toMap(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("toMap: %v", err)
	}
	return m
}

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name    string
		base    map[string]any
		overlay map[string]any
		want    map[string]any
	}{
		{
			name:    "empty base + overlay with key",
			base:    map[string]any{},
			overlay: map[string]any{"foo": "bar"},
			want:    map[string]any{"foo": "bar"},
		},
		{
			name:    "base with key + empty overlay",
			base:    map[string]any{"foo": "bar"},
			overlay: map[string]any{},
			want:    map[string]any{"foo": "bar"},
		},
		{
			name:    "scalar override",
			base:    map[string]any{"foo": "original"},
			overlay: map[string]any{"foo": "overridden"},
			want:    map[string]any{"foo": "overridden"},
		},
		{
			name:    "nested object merge preserves base key",
			base:    toMap(t, `{"sandbox":{"enabled":true}}`),
			overlay: toMap(t, `{"sandbox":{"allowWrite":"/tmp"}}`),
			want:    toMap(t, `{"sandbox":{"enabled":true,"allowWrite":"/tmp"}}`),
		},
		{
			name:    "array replacement not merge",
			base:    map[string]any{"items": []any{"a", "b"}},
			overlay: map[string]any{"items": []any{"c"}},
			want:    map[string]any{"items": []any{"c"}},
		},
		{
			name:    "new keys added alongside existing",
			base:    map[string]any{"a": 1},
			overlay: map[string]any{"b": 2},
			want:    map[string]any{"a": 1, "b": 2},
		},
		{
			name:    "deeply nested 3+ levels",
			base:    toMap(t, `{"a":{"b":{"c":"base","d":"keep"}}}`),
			overlay: toMap(t, `{"a":{"b":{"c":"overlay"}}}`),
			want:    toMap(t, `{"a":{"b":{"c":"overlay","d":"keep"}}}`),
		},
		{
			name:    "overlay scalar replaces base object",
			base:    toMap(t, `{"key":{"nested":"value"}}`),
			overlay: map[string]any{"key": "scalar"},
			want:    map[string]any{"key": "scalar"},
		},
		{
			name:    "overlay object replaces base scalar",
			base:    map[string]any{"key": "scalar"},
			overlay: toMap(t, `{"key":{"nested":"value"}}`),
			want:    toMap(t, `{"key":{"nested":"value"}}`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := settings.DeepMerge(tc.base, tc.overlay)

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("DeepMerge() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestDeepMergeDoesNotModifyInputs(t *testing.T) {
	base := map[string]any{"a": map[string]any{"x": 1}}
	overlay := map[string]any{"a": map[string]any{"y": 2}}

	baseJSON, _ := json.Marshal(base)
	overlayJSON, _ := json.Marshal(overlay)

	settings.DeepMerge(base, overlay)

	baseAfterJSON, _ := json.Marshal(base)
	overlayAfterJSON, _ := json.Marshal(overlay)

	if string(baseJSON) != string(baseAfterJSON) {
		t.Errorf("DeepMerge modified base: before=%s after=%s", baseJSON, baseAfterJSON)
	}
	if string(overlayJSON) != string(overlayAfterJSON) {
		t.Errorf("DeepMerge modified overlay: before=%s after=%s", overlayJSON, overlayAfterJSON)
	}
}
