package creds

import (
	"awsso/pkg/cli/profiles"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type CredsOptions struct {
	profile string
	login   bool
}

func credentialsFile() (*ini.File, error) {
	dirname, err := os.UserHomeDir()

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	credsParth := path.Join(dirname, ".aws/credentials")
	cfg, err := ini.Load(credsParth)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return cfg, nil
}

func NewCredsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "creds PROFILE",
		Short: "Refresh short-term credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := args[0]
			login, err := cmd.Flags().GetBool("login")

			if err != nil {
				log.Fatal(err)
				return err
			}

			opts := CredsOptions{
				profile: profile,
				login:   login,
			}

			return runCreds(opts)
		},
	}

	cmd.Flags().Bool("login", false, "Creates an AWS SSO login session before fetching credentials")

	return cmd
}

func runCreds(opts CredsOptions) error {
	if opts.login {
		cmd := exec.Command("aws", "sso", "login", "--profile", opts.profile)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Fatal(err)
			return err
		}
	}

	configs, err := profiles.ConfigsMap()

	if err != nil {
		log.Fatal(err)
		return err
	}

	section, ok := configs["profile "+opts.profile]

	if !ok {
		return fmt.Errorf("awsso: profile not found")
	}

	ssoCache, err := section.Profile.SsoCache()

	if err != nil {
		log.Fatal(err)
		return err
	}

  expiresAt, err := time.Parse(time.RFC3339, ssoCache.ExpiresAt)

	if err != nil {
		log.Fatal(err)
		return err
	}

  if time.Now().After(expiresAt) {
    return fmt.Errorf("awsso: sso credentials have expired, please re-run using '--login'")
  }

	fmt.Printf("%+v\n", opts)
	return nil
}

