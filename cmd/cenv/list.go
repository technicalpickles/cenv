package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := env.List()
		if err != nil {
			return err
		}

		if listJSON {
			infos := make([]*env.Info, 0, len(names))
			for _, name := range names {
				info, err := env.Inspect(name)
				if err != nil {
					return fmt.Errorf("inspecting %q: %w", name, err)
				}
				infos = append(infos, info)
			}
			out, err := json.MarshalIndent(infos, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Println(string(out))
			return nil
		}

		if len(names) == 0 {
			fmt.Println("No environments yet.")
			fmt.Println("Create one: cenv create <name>")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tAUTH")
		for _, name := range names {
			status := "no"
			if auth.Detect(env.Path(name)) == nil {
				status = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\n", name, status)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Emit environments as JSON with metadata")
	rootCmd.AddCommand(listCmd)
}
