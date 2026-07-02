package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an environment",
	Example: `  cenv remove myenv
  cenv remove myenv --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !removeForce && !confirmRemoval(name) {
			fmt.Println("Aborted.")
			return nil
		}

		if err := env.Remove(name); err != nil {
			return err
		}
		logf("[cenv] Removed environment %q\n", name)
		return nil
	},
}

// confirmRemoval prompts for confirmation when stdin is a terminal.
// In non-interactive contexts (scripts, CI, piped input) it returns true
// immediately rather than blocking on input that will never arrive.
func confirmRemoval(name string) bool {
	if !isTerminal(os.Stdin) {
		return true
	}
	fmt.Fprintf(os.Stderr, "Remove environment %q? [y/N] ", name)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return parseConfirmation(line)
}

// parseConfirmation reports whether input is an affirmative response
// ("y" or "yes", case-insensitive, surrounding whitespace ignored).
func parseConfirmation(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func init() {
	removeCmd.Flags().BoolVar(&removeForce, "force", false, "Skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}
