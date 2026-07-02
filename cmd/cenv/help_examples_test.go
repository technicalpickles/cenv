package main

import "testing"

func TestHelpExamples_Present(t *testing.T) {
	cases := map[string]string{
		"create":         createCmd.Example,
		"run":            runCmd.Example,
		"login":          loginCmd.Example,
		"remove":         removeCmd.Example,
		"path":           pathCmd.Example,
		"trust":          trustCmd.Example,
		"settings show":  settingsShowCmd.Example,
		"settings get":   settingsGetCmd.Example,
		"settings merge": settingsMergeCmd.Example,
	}
	for name, example := range cases {
		if example == "" {
			t.Errorf("%s: missing Example text", name)
		}
	}
}
