package main

import "testing"

func TestColorEnabled(t *testing.T) {
	cases := []struct {
		name          string
		noColorFlag   bool
		noColorEnvSet bool
		stdoutIsTTY   bool
		want          bool
	}{
		{"tty, nothing disabled", false, false, true, true},
		{"--no-color set", true, false, true, false},
		{"NO_COLOR set", false, true, true, false},
		{"stdout not a tty", false, false, false, false},
		{"everything disabled", true, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := colorEnabled(tc.noColorFlag, tc.noColorEnvSet, tc.stdoutIsTTY)
			if got != tc.want {
				t.Errorf("colorEnabled(%v, %v, %v) = %v, want %v",
					tc.noColorFlag, tc.noColorEnvSet, tc.stdoutIsTTY, got, tc.want)
			}
		})
	}
}
