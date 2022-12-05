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

	// todo show available versions (create secret first!)
	if config.Version == "" {
		prompt = &survey.Input{
			Message: "Which version of 23ke would you like to install (should match a git tag)?",
		}
		err = survey.AskOne(prompt, &config.Version, withValidator("required"))
		handleErr(err)
	}

	// todo explain to user. what's this for?
	if config.EmailAddress == "" {
		prompt = &survey.Input{
			Message: "Please enter your email address",
		}
		err = survey.AskOne(prompt, &config.EmailAddress, withValidator("required,email"))
		handleErr(err)
	}

	// todo explain to user. what's this for?
	if config.Issuer.Acme.Email == "" {
		prompt = &survey.Input{
			Message: "Please enter your email address for acme certificate generation",
			Default: config.EmailAddress,
		}
		err = survey.AskOne(prompt, &config.Issuer.Acme.Email, withValidator("required,email"))
		handleErr(err)
	}
	// todo move right after user emailaddress
	// todo explain to user. what's this for?
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

	// todo explain to user. what's this for?
	if config.GitRepo == "" {
		prompt = &survey.Input{
			// todo allow form git@github.com:User/Repo.git and transform it to url form.
			// todo don't allow http url
			Message: "Please enter your git repository remote, e.g. ssh://git@github.com/User/Repo.git",
		}
		err = survey.AskOne(prompt, &config.GitRepo, withValidator("required,url"))
		handleErr(err)
	}

	// todo explain to user. what's this for?
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

	// todo explain to user. document where to find it on supported providers
	if config.BaseCluster.NodeCidr == "" {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
		}
		err = survey.AskOne(prompt, &config.BaseCluster.NodeCidr, withValidator("required,cidr"))
		handleErr(err)
	}
	config.Gardenlet.SeedNodeCidr = config.BaseCluster.NodeCidr

	// todo explain to user. what does "I don't know" imply?
	if config.BaseCluster.HasVerticalPodAutoscaler == nil {
		const (
			yes       = "Yes"
			no        = "No"
			iDontKnow = "I don't know"
		)

		prompt = &survey.Select{
			Message: "Does your base cluster provide vertical pod autoscaling (VPA)?",
			Options: []string{yes, no, iDontKnow},
		}

		var answer string

		err = survey.AskOne(prompt, &answer)
		handleErr(err)

		var hasVerticalPodAutoscaler bool

		switch answer {
		case yes:
			hasVerticalPodAutoscaler = true
		case no:
			hasVerticalPodAutoscaler = false
		case iDontKnow:
			hasVerticalPodAutoscaler = false

			printWarn(`A Vertical Pod Autoscaler will be deployed. If the base cluster already provides one, both may keep reversing the other one's changes. Gardener will work but you'll see lots of pod restarts. Not recommended for production use.`)
			pressEnterToContinue()
		}

		config.BaseCluster.HasVerticalPodAutoscaler = &hasVerticalPodAutoscaler
	}

	if config.DomainConfig.Domain == "" || config.DomainConfig.Provider == "" {
		config.DomainConfig = queryDomainConfig()
	}

}

// todo explain to user. ask for domain after provider configuration to make it clearer what this domain is meant for.
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
		Options: []string{DNS_PROVIDER_AZURE_DNS, DNS_PROVIDER_OPENSTACK_DESIGNATE, DNS_PROVIDER_AWS_ROUTE_53},
	}
	err = survey.AskOne(prompt, &provider, withValidator("required"))
	handleErr(err)

	domainConfig, _ := createDomainConfiguration(domain, provider)
	return domainConfig
}

func (d *dnsCredentialsAzure) parseCredentials() {
	qs := []*survey.Question{
		{
			Name:      "TenantId",
			Prompt:    &survey.Input{Message: "Azure tenant ID? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "SubscriptionId",
			Prompt:    &survey.Input{Message: "Azure subscription ID? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "ClientID",
			Prompt:    &survey.Input{Message: "Azure client ID? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "ClientSecret",
			Prompt:    &survey.Input{Message: "Azure client secret? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	handleErr(err)
}

func (d *dnsCredentialsOSDesignate) parseCredentials() {
	qs := []*survey.Question{
		{
			Name:      "ApplicationCredentialID",
			Prompt:    &survey.Input{Message: "Application Credential ID? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "ApplicationCredentialSecret",
			Prompt:    &survey.Input{Message: "Application Credential Secret? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "AuthURL",
			Prompt:    &survey.Input{Message: "AuthURL? (plain or base64)"},
			Validate:  makeValidator("required,url"),
			Transform: survey.TransformString(coerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	handleErr(err)
}

func (d *dnsCredentialsAWS53) parseCredentials() {
	qs := []*survey.Question{
		{
			Name:      "AccessKeyID",
			Prompt:    &survey.Input{Message: "Access Key ID? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
		{
			Name:      "SecretAccessKey",
			Prompt:    &survey.Input{Message: "Secret Access Key? (plain or base64)"},
			Validate:  makeValidator("required"),
			Transform: survey.TransformString(coerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	handleErr(err)
}

func withValidator(tag string) survey.AskOpt {
	return survey.WithValidator(makeValidator(tag))
}
