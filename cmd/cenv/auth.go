package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/claudeauth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)

var authClient = claudeauth.Default

var authStatusLive bool
var authStatusText bool

const probeTimeout = 60 * time.Second

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Check environment authentication",
}

var authStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show auth status for an environment",
	Example: `  cenv auth status myenv
  cenv auth status myenv --live`,
	Long: `Runs 'claude auth status' inside the named environment and passes its
output through. Exits nonzero if no credentials are stored.

'claude auth status' only reports what's stored; it doesn't check expiry or
talk to the network, so an expired login still shows as logged in. Pass
--live to also make one minimal real request (Haiku, no tools, safe mode,
no session saved; about $0.002) and fail if it's rejected.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}
		envDir := env.Path(name)

		format := "--json"
		if authStatusText {
			format = "--text"
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		st, err := authClient.Status(ctx, envDir, format)
		if err != nil {
			return err
		}
		os.Stdout.Write(st.Output)
		if !st.LoggedIn {
			return fmt.Errorf("env %q is not logged in; run 'cenv login %s'", name, name)
		}

		if !authStatusLive {
			return nil
		}

		ctx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		res, err := authClient.Probe(ctx, envDir)
		if err != nil {
			return fmt.Errorf("live check for %q couldn't run: %w", name, err)
		}
		if !res.OK {
			return fmt.Errorf("live check for %q failed: %s\nrun 'cenv login %s' to re-authenticate", name, res.Message, name)
		}

		logf("%s\n", style.Success("Live check passed for %q ($%.4f)", name, res.CostUSD))
		return nil
	},
}

func init() {
	authStatusCmd.Flags().BoolVar(&authStatusLive, "live", false, "Also make a minimal real request to verify the credentials work")
	authStatusCmd.Flags().BoolVar(&authStatusText, "text", false, "Human-readable output instead of JSON")
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
