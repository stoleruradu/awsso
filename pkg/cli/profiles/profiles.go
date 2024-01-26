package profiles

import (
	"awsso/pkg/printer"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type ConfigSection struct {
	Name    string
	Profile ConfigProfile
}

func (s *ConfigSection) ShortName() string {
	split := strings.Split(s.Name, " ")
	return split[len(split)-1]
}

type ConfigProfile struct {
	region       string
	SsoAccountId string
	SsoRoleName  string
	ssoStartUrl  string
	SsoRegion    string
}

type SsoCache struct {
	AccessToken string `json:"accessToken"`
  ExpiresAt   string `json:"expiresAt"`
}

func (s ConfigProfile) SsoCache() (*SsoCache, error) {
	h := sha1.New()
	h.Write([]byte(s.ssoStartUrl))

	sha1Hex := hex.EncodeToString(h.Sum(nil))

	dirname, err := os.UserHomeDir()

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	cachePath := path.Join(dirname, ".aws", "sso", "cache", sha1Hex + ".json")
	dat, err := os.ReadFile(cachePath)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var cache SsoCache

  if err := json.Unmarshal(dat, &cache); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &cache, nil
}

func ConfigsMap() (map[string]*ConfigSection, error) {
	dirname, err := os.UserHomeDir()

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	cfgPath := path.Join(dirname, ".aws/config")
	cfg, err := ini.Load(cfgPath)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	sections := cfg.Sections()
	configs := make(map[string]*ConfigSection)

	for _, section := range sections {
		if len(section.Keys()) == 0 {
			continue
		}

		keysHash := section.KeysHash()

		configs[section.Name()] = &ConfigSection{
			Name: section.Name(),
			Profile: ConfigProfile{
				region:       keysHash["region"],
				SsoAccountId: keysHash["sso_account_id"],
				SsoRoleName:  keysHash["sso_role_name"],
				ssoStartUrl:  keysHash["sso_start_url"],
				SsoRegion:    keysHash["sso_region"],
			},
		}
	}

	return configs, nil
}

type ProfileListItem struct {
	name   string
	role   string
	region string
}

func NewProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "List available sso profiles",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			hashMap, err := ConfigsMap()

			if err != nil {
				log.Fatal(err)
				return errors.New("awsso: failed to list profiles. Try '--verbose' for more info")
			}

			profiles := make([]ProfileListItem, len(hashMap))

			var i int
			for _, section := range hashMap {
				shortName := section.ShortName()
				role := section.Profile.SsoRoleName
				region := section.Profile.region

				profiles[i] = ProfileListItem{
					name:   shortName,
					role:   role,
					region: region,
				}

				i += 1
			}

			printer.Table(profiles)

			return nil
		},
	}

	return cmd
}
