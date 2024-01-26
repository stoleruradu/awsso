package creds

import (
	"awsso/pkg/cli/profiles"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type CredsOptions struct {
	profile string
	dryRun  bool
	login   bool
	backup  bool
}

type CredsFile struct {
	path       string
	descriptor *ini.File
}

func credentialsFile() (*CredsFile, error) {
	dirname, err := os.UserHomeDir()

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	credsPath := path.Join(dirname, ".aws/credentials")
	cfg, err := ini.Load(credsPath)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &CredsFile{descriptor: cfg, path: credsPath}, nil
}

func NewCredsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "creds PROFILE",
		Short: "Refresh short-term credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := args[0]
			login, _ := cmd.Flags().GetBool("login")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			backup, _ := cmd.Flags().GetBool("backup")

			opts := CredsOptions{
				profile: profile,
				login:   login,
				dryRun:  dryRun,
				backup:  backup,
			}

			return runCreds(opts)
		},
	}

	cmd.Flags().Bool("login", false, "Creates an AWS SSO login session before fetching credentials")
	cmd.Flags().Bool("dry-run", false, "Writes to stdout")
	cmd.Flags().Bool("backup", false, "Makes a buckup before writing to credentials file")

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

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatal(err)
		return err
	}

	cfg.Region = section.Profile.SsoRegion

	client := sso.NewFromConfig(cfg)

	params := sso.GetRoleCredentialsInput{
		AccountId:   &section.Profile.SsoAccountId,
		RoleName:    &section.Profile.SsoRoleName,
		AccessToken: &ssoCache.AccessToken,
	}

	roleCreds, err := client.GetRoleCredentials(context.TODO(), &params)

	if err != nil {
		log.Fatal(err)
		return err
	}

	credFile, err := credentialsFile()

	if err != nil {
		log.Fatal(err)
		return err
	}

	if opts.backup {
		f, err := os.Create(credFile.path + ".bak")
		defer f.Close()

		if err != nil {
			log.Fatal(err)
			return err
		}

		w := bufio.NewWriter(f)
		credFile.descriptor.WriteTo(w)

    err = w.Flush()

		if err != nil {
			log.Fatal(err)
			return err
		}
	}

	for _, sec := range credFile.descriptor.Sections() {
		if sec.Name() == section.ShortName() {
			sec.Key("region").SetValue(cfg.Region)
			sec.Key("aws_access_key_id").SetValue(*roleCreds.RoleCredentials.AccessKeyId)
			sec.Key("aws_secret_access_key").SetValue(*roleCreds.RoleCredentials.SecretAccessKey)
			sec.Key("aws_session_token").SetValue(*roleCreds.RoleCredentials.SessionToken)
		}
	}

	if opts.dryRun {
		credFile.descriptor.WriteTo(os.Stdout)
		return nil
	}

	credFile.descriptor.SaveTo(credFile.path)
	return nil
}
