package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

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
	ssoAccountId string
	ssoRoleName  string
	ssoStartUrl  string
	ssoRegion    string
}

func configs() (map[string]*ConfigSection, error) {
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
				ssoAccountId: keysHash["sso_account_id"],
				ssoRoleName:  keysHash["sso_role_name"],
				ssoStartUrl:  keysHash["sso_start_url"],
				ssoRegion:    keysHash["sso_region"],
			},
		}
	}

	return configs, nil
}

func profilesCommand() error {
	hashMap, err := configs()

	if err != nil {
		return err
	}

	for _, profile := range hashMap {
		fmt.Println(profile.ShortName())
	}

	return nil
}

func main() {
	err := profilesCommand()

	if err != nil {
		os.Exit(1)
	}
}
