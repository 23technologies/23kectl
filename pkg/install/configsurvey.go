package install

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

func handleErr(err error) {
	if errors.Is(err, terminal.InterruptErr) {
		fmt.Println("Ctrl+C, exiting.")
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}
}

// queryAdminConfig ...
func queryAdminConfig()  {
	var err error
	var prompt survey.Prompt

	if !viper.IsSet("admin.email") {
		prompt = &survey.Input{
			Message: "Please enter your email address",
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,email"))
		handleErr(err)
		viper.Set("admin.email", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.password") {
		prompt = &survey.Password{
			Message: "Please enter the administrator password to use",
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		handleErr(err)

		hash, err := bcrypt.GenerateFromPassword(([]byte)(queryResult), 10)
		handleErr(err)
		viper.Set("admin.password", string(hash))
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.gitrepourl") {
		prompt = &survey.Input{
			Message: "Please enter an ssh git remote in URL form. e.g. ssh://git@github.com/User/Repo.git",
			Help: `Configuration files are to be stored in this repo. Flux will monitor these files to pick up configuration changes.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,url,startswith=ssh://"))
		handleErr(err)
		viper.Set("admin.gitrepourl", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.gitrepobranch") {
		prompt = &survey.Input{
			Message: "Please enter the git branch to use. Will be created if it doesn't exist.",
			Default: "main",
			Help: `Can be any branch name you want. You can store configuration files for multiple gardeners (e.g. prod, staging, dev) on the same repo by choosing unique branch names for them.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		handleErr(err)
		viper.Set("admin.gitrepobranch", queryResult)
		viper.WriteConfig()
	}
}

func queryBaseClusterConfig() {
	var err error
	var prompt survey.Prompt

	// todo explain to user. what's this for?
	if !viper.IsSet("baseCluster.provider") {
		prompt = &survey.Select{
			Message: "Select the provider of your base cluster",
			Options: []string{"hcloud", "azure", "aws", "openstack"},
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		handleErr(err)
		viper.Set("baseCluster.provider", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("baseCluster.Region") {
		prompt = &survey.Input{
			Message: "Please enter the region of your base cluster",
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		handleErr(err)
		viper.Set("baseCluster.region", queryResult)
		viper.WriteConfig()
	}

	// todo explain to user. document where to find it on supported providers
	if !viper.IsSet("baseCluster.nodeCidr") {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,cidr"))
		handleErr(err)
		viper.Set("baseCluster.nodeCidr", queryResult)
	}
	viper.Set("gardenlet.seedNodeCidr", viper.GetString("baseCluster.nodeCidr"))
	viper.WriteConfig()

	if !viper.IsSet("baseCluster.hasVerticalPodAutoscaler") {
		const (
			yes       = "Yes"
			no        = "No"
			iDontKnow = "I don't know"
		)

		prompt = &survey.Select{
			Message: "Does your base cluster provide vertical pod autoscaling (VPA)?",
			Options: []string{yes, no, iDontKnow},
			Help: `Depending on your provider and setup, your base cluster may or may not provide this functionality. If it doesn't, we'll install everything necessary for gardener to work.
Automatically detecting VPA from within the cluster isn't reliable, so if you choose "I don't know" a VPA is installed just in case. You might end up with two autoscalers, which will generally work for evaluation but causes unexpected behavior like very frequent pod restarts`,
		}

		var queryResult string

		err = survey.AskOne(prompt, &queryResult)
		handleErr(err)

		var hasVerticalPodAutoscaler bool

		switch queryResult {
		case yes:
			hasVerticalPodAutoscaler = true
		case no:
			hasVerticalPodAutoscaler = false
		case iDontKnow:
			hasVerticalPodAutoscaler = false

			printWarn(`A Vertical Pod Autoscaler will be deployed.
If the base cluster already provides one, both may keep reversing the other one's changes.
Gardener will work but you'll see lots of pod restarts. Not recommended for production use.`)
			pressEnterToContinue()
		}
		viper.Set("baseCluster.hasVerticalPodAutoscaler", hasVerticalPodAutoscaler)
		viper.WriteConfig()
	}


}

func queryDomainConfig() domainConfiguration {
	var err error
	var domain, provider string
	var prompt survey.Prompt

	prompt = &survey.Input{
		Message: "Please enter the base (sub)domain of your gardener setup. Gardener components will be available as subdomains of this (e.g dashboard.<gardener.my-company.io>). Has to be configurable through one of the supported DNS providers.",
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

