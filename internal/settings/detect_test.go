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
		{`config.json`, false},
		{`/path/to/file.json`, false},
		{``, false},
		{`not json at all`, false},
		{`true`, false},
	}

	for _, tt := range tests {
		got := IsJSON(tt.input)
		if got != tt.want {
			t.Errorf("IsJSON(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
