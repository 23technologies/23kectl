package install

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/crypto/bcrypt"
	"os"
)

func handleErr(err error) {
	if errors.Is(err, terminal.InterruptErr) {
		fmt.Println("Ctrl+C, exiting.")
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}
}

func queryConfig(config *KeConfig) {
	var err error
	var prompt survey.Prompt

	if config.EmailAddress == "" {
		prompt = &survey.Input{
			Message: "Please enter your email address",
		}
		err = survey.AskOne(prompt, &config.EmailAddress, withValidator("required,email"))
		handleErr(err)
		config.Issuer.Acme.Email = config.EmailAddress
	}

	if config.AdminPassword == "" {
		var plainPassword string

		prompt = &survey.Password{
			Message: "Please enter the administrator password to use",
		}
		err = survey.AskOne(prompt, &plainPassword, withValidator("required"))
		handleErr(err)

		hash, err := bcrypt.GenerateFromPassword(([]byte)(plainPassword), 10)
		config.AdminPassword = string(hash)
		handleErr(err)
	}

	if config.GitRepo == "" {
		prompt = &survey.Input{
			Message: "Please enter your git repository remote, e.g. git@github.com:User/Repo.git",
		}
		err = survey.AskOne(prompt, &config.GitRepo, withValidator("required"))
		handleErr(err)
	}

	if config.BaseCluster.Provider == "" {
		prompt = &survey.Select{
			Message: "Select the provider of your base cluster",
			Options: []string{"hcloud", "azure", "aws", "openstack"},
		}
		err = survey.AskOne(prompt, &config.BaseCluster.Provider, withValidator("required"))
		handleErr(err)
	}

	if config.BaseCluster.Region == "" {
		prompt = &survey.Input{
			Message: "Please enter the region of your base cluster",
		}
		err = survey.AskOne(prompt, &config.BaseCluster.Region, withValidator("required"))
		handleErr(err)
	}

	if config.BaseCluster.NodeCidr == "" {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
		}
		err = survey.AskOne(prompt, &config.BaseCluster.NodeCidr, withValidator("required,cidr"))
		handleErr(err)
		config.Gardenlet.SeedNodeCidr = config.BaseCluster.NodeCidr
	}

	if config.DomainConfig.Domain == "" || config.DomainConfig.Provider == "" {
		config.DomainConfig = queryDomainConfig()
	}

}

func queryDomainConfig() domainConfiguration {
	var err error
	var domain, provider string
	var prompt survey.Prompt

	prompt = &survey.Input{
		Message: "Please enter your domain",
	}
	err = survey.AskOne(prompt, &domain, withValidator("required,fqdn"))
	handleErr(err)

	prompt = &survey.Select{
		Message: "Define your DNS provider",
		Options: []string{"azure-dns", "openstack-designate", "aws-route53"},
	}
	err = survey.AskOne(prompt, &provider, withValidator("required"))
	handleErr(err)

	domainConfig, _ := createDomainConfiguration(domain, provider)
	return domainConfig
}

func (d *dnsCredentialsAzure) parseCredentials() {
	qs := []*survey.Question{
		{
			Name:     "TenantId",
			Prompt:   &survey.Input{Message: "Azure tenant ID?"},
			Validate: makeValidator("required"),
		},
		{
			Name:     "SubscriptionId",
			Prompt:   &survey.Input{Message: "Azure subscription ID?"},
			Validate: makeValidator("required"),
		},
		{
			Name:     "SecretId",
			Prompt:   &survey.Input{Message: "Azure secret ID?"},
			Validate: makeValidator("required"),
		},
		{
			Name:     "SecretValue",
			Prompt:   &survey.Input{Message: "Azure subscription ID?"},
			Validate: makeValidator("required"),
		},
	}

	err := survey.Ask(qs, d)
	handleErr(err)
}

func withValidator(tag string) survey.AskOpt {
	return survey.WithValidator(makeValidator(tag))
}
