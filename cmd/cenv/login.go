package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)

var loginCmd = &cobra.Command{
	Use:   "login <name> [-- claude-auth-login-flags...]",
	Short: "Log in to Claude in an environment (claude auth login)",
	Example: `  cenv login myenv
  cenv login myenv -- --sso
  cenv login myenv -- --console
  CENV_LOGIN_FLAGS=--sso cenv login myenv`,
	Long: `Runs 'claude auth login' with CLAUDE_CONFIG_DIR pointed at the named
environment. Claude prints a sign-in URL and waits for the code on stdin, so
no REPL or PTY is needed.

With no flags after --, cenv runs plain 'claude auth login' (Claude
subscription), or passes the flags in CENV_LOGIN_FLAGS if set (for example
CENV_LOGIN_FLAGS=--sso to always use SSO). Anything after -- replaces that
default and goes straight to 'claude auth login'.

Stdin, stdout and stderr are passed through. To script it, run cenv login with
a pipe on stdin, read the URL from its output, and write the code back.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		argv, environ := loginExec(env.Path(name), args[1:], os.Environ())

		logf("%s\n", style.Info("Running claude auth login in %q.", name))

		return syscall.Exec(claudePath, argv, environ)
	},
}

// loginExec builds the argv and environment for 'claude auth login'. With no
// extra flags it falls back to the whitespace-separated flags in
// CENV_LOGIN_FLAGS, or none.
func loginExec(envDir string, extra, environ []string) (argv, newEnviron []string) {
	if len(extra) == 0 {
		extra = strings.Fields(os.Getenv("CENV_LOGIN_FLAGS"))
	}
	argv = append([]string{"claude", "auth", "login"}, extra...)
	newEnviron = append(append([]string{}, environ...), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))
	return argv, newEnviron
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
