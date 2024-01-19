package main

import (
	"awsso/pkg/cli/profiles"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var AwssoCommand = &cobra.Command{
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
  AwssoCommand.SetHelpCommand(&cobra.Command{Hidden: true})
  AwssoCommand.AddCommand(profiles.NewProfilesCommand())
}

func main() {
	if err := AwssoCommand.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
