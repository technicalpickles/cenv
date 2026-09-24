package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)

var authRefreshAll bool

var authRefreshCmd = &cobra.Command{
	Use:   "refresh [name...]",
	Short: "Make a live request in each environment to keep its login warm",
	Example: `  cenv auth refresh myenv
  cenv auth refresh --all`,
	Long: `Runs the same minimal live request as 'cenv auth status --live' in each
named environment (or every environment with --all). Claude Code refreshes
an env's OAuth token during a request once the access token is expired or
within 5 minutes of expiring, so running this daily refreshes envs you
rarely launch and tells you which ones have already gone stale.

Environments with nothing stored are skipped. Exits nonzero if any env's
live request is rejected, after checking the rest.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := refreshTargets(args)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		var dead []string
		for _, name := range names {
			if !refreshEnv(ctx, name) {
				dead = append(dead, name)
			}
		}

		if len(dead) > 0 {
			hints := make([]string, len(dead))
			for i, name := range dead {
				hints[i] = fmt.Sprintf("cenv login %s", name)
			}
			return fmt.Errorf("%d env(s) failed the live check; run: %s", len(dead), strings.Join(hints, ", "))
		}
		return nil
	},
}

func refreshTargets(args []string) ([]string, error) {
	switch {
	case authRefreshAll && len(args) > 0:
		return nil, fmt.Errorf("pass env names or --all, not both")
	case authRefreshAll:
		return env.List()
	case len(args) == 0:
		return nil, fmt.Errorf("pass one or more env names, or --all")
	}
	for _, name := range args {
		if !env.Exists(name) {
			return nil, fmt.Errorf("environment %q not found", name)
		}
	}
	return args, nil
}

// refreshEnv returns true for skipped envs; only stored-but-rejected
// credentials (or a probe that can't run) count as failures.
func refreshEnv(ctx context.Context, name string) bool {
	envDir := env.Path(name)

	st, err := authClient.Status(ctx, envDir, "--json")
	if err != nil {
		fmt.Println(style.Error("%s: couldn't check status: %v", name, err))
		return false
	}
	if !st.LoggedIn {
		fmt.Println(style.Warning("%s: skipped, not logged in", name))
		return true
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	res, err := authClient.Probe(probeCtx, envDir)
	switch {
	case err != nil:
		fmt.Println(style.Error("%s: live check couldn't run: %v", name, err))
		return false
	case !res.OK:
		fmt.Println(style.Error("%s: %s", name, res.Message))
		return false
	}

	fmt.Println(style.Success("%s: ok ($%.4f)", name, res.CostUSD))
	return true
}

func init() {
	authRefreshCmd.Flags().BoolVar(&authRefreshAll, "all", false, "Refresh every environment")
	authCmd.AddCommand(authRefreshCmd)
}
