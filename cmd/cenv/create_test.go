package main

import (
	"strings"
	"testing"
)

func TestCreateCmd_BareAndFromMutuallyExclusive(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	createBare = true
	createFrom = "user"
	t.Cleanup(func() {
		createBare = false
		createFrom = ""
	})

	err := createCmd.RunE(createCmd, []string{"myenv"})
	if err == nil {
		t.Fatal("expected error for --bare + --from, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %q, want it to mention 'mutually exclusive'", err.Error())
	}
}
