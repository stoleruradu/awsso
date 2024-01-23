package main

import (
	"awsso/pkg/cli/creds"
	"awsso/pkg/cli/profiles"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:              "awsso [OPTIONS] COMMAND [ARG...]",
	Short:            "AWS sso helper",
	SilenceUsage:     true,
	SilenceErrors:    true,
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		return fmt.Errorf("awsso: '%s' is not a valid command.\nSee 'awsso --help'", args[0])
	},
	DisableFlagsInUseLine: true,
}

func init() {
  root.CompletionOptions.DisableDefaultCmd = true
  root.DisableFlagsInUseLine = true
  root.SetHelpCommand(&cobra.Command{Hidden: true})
  root.AddCommand(creds.NewCredsCommand())
  root.AddCommand(profiles.NewProfilesCommand())
}

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
