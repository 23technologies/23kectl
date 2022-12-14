package install

import (
	"errors"
	"fmt"
	"github.com/23technologies/23kectl/pkg/common"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

func exitOnCtrlC(err error) {
	if errors.Is(err, terminal.InterruptErr) {
		fmt.Println("Ctrl+C, exiting.")
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}
}

// queryAdminConfig ...
func queryAdminConfig() error {
	var err error
	var prompt survey.Prompt

	if !viper.IsSet("admin.email") {
		prompt = &survey.Input{
			Message: `Please enter your email address.
This will be the email address to use, when you want to login to the Gardener dashboard.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,email"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("admin.email", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.password") {
		prompt = &survey.Password{
			Message: `Please enter the administrator password to use.
This will be the password to use, when you login to the Gardener dashboard.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword(([]byte)(queryResult), 10)
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("admin.password", string(hash))
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.gitrepourl") {
		prompt = &survey.Input{
			Message: "Please enter an ssh git remote in URL form. e.g. ssh://git@github.com/User/Repo.git",
			Help: `
Configuration files are to be stored in this repo.
Flux will monitor these files to pick up configuration changes.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,url,startswith=ssh://"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("admin.gitrepourl", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("admin.gitrepobranch") {
		prompt = &survey.Input{
			Message: "Please enter the git branch to use. Will be created if it doesn't exist.",
			Default: "main",
			Help: `
Can be any branch name you want.
You can store configuration files for multiple gardeners (e.g. prod, staging, dev) on the same repo by choosing unique branch names for them.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("admin.gitrepobranch", queryResult)
		viper.WriteConfig()
	}

	return nil
}

func queryBaseClusterConfig() error {
	var err error
	var prompt survey.Prompt

	// todo explain to user. what's this for?
	if !viper.IsSet("baseCluster.provider") {
		prompt = &survey.Select{
			Message: "Select the provider of your base cluster",
			Options: []string{"hcloud", "azure", "aws", "openstack"},
			Help: `
Currently, this tools supports the listed providers for base clusters.
If you feel like this list in incomplete, contact the 23T support.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("baseCluster.provider", queryResult)
		viper.WriteConfig()
	}

	if !viper.IsSet("baseCluster.Region") {
		prompt = &survey.Input{
			Message: "Please enter the region of your base cluster",
			Help: `
This is the region your base cluster runs in.
Generally this is dependent on the provider of your base cluster.
For clusters hosted on Azure, this could be e.g. germanywestcentral or westeurope.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("baseCluster.region", queryResult)
		viper.WriteConfig()
	}

	// todo explain to user. document where to find it on supported providers
	if !viper.IsSet("baseCluster.nodeCidr") {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
			Help: `
Gardener will check whether the nodes' ip addresses of your base cluster lie in the specified network.
Therefore, the node CIDR should match a network that comprises all ip addresses of your nodes.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required,cidr"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
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
			Help: `
Depending on your provider and setup, your base cluster may or may not provide this functionality.
If it doesn't, we'll install everything necessary for gardener to work.
Automatically detecting VPA from within the cluster isn't reliable, so if you choose "I don't know" a VPA is installed just in case. You might end up with two autoscalers, which will generally work for evaluation but causes unexpected behavior like very frequent pod restarts
`,
		}

		var queryResult string

		err = survey.AskOne(prompt, &queryResult)
		exitOnCtrlC(err)
		if err != nil {
			return err
		}

		var hasVerticalPodAutoscaler bool

		switch queryResult {
		case yes:
			hasVerticalPodAutoscaler = true
		case no:
			hasVerticalPodAutoscaler = false
		case iDontKnow:
			hasVerticalPodAutoscaler = false

			common.PrintWarn(`A Vertical Pod Autoscaler will be deployed.
If the base cluster already provides one, both may keep reversing the other one's changes.
Gardener will work but you'll see lots of pod restarts. Not recommended for production use.`)
			common.PressEnterToContinue()
		}
		viper.Set("baseCluster.hasVerticalPodAutoscaler", hasVerticalPodAutoscaler)
		viper.WriteConfig()
	}
	return nil
}

func queryDomainConfig() (*domainConfiguration, error) {
	var err error
	var domain, provider string
	var prompt survey.Prompt

	prompt = &survey.Input{
		Message: `Please enter the base (sub)domain of your gardener setup.
Gardener components will be available as subdomains of this (e.g dashboard.<gardener.my-company.io>).
Note that it has to be delegated to the chosen DNS provider.`,
	}
	err = survey.AskOne(prompt, &domain, withValidator("required,fqdn"))
	exitOnCtrlC(err)
	if err != nil {
		return nil, err
	}

	prompt = &survey.Select{
		Message: "Define your DNS provider",
		Options: []string{common.DNS_PROVIDER_AZURE_DNS, common.DNS_PROVIDER_OPENSTACK_DESIGNATE, common.DNS_PROVIDER_AWS_ROUTE_53},
	}
	err = survey.AskOne(prompt, &provider, withValidator("required"))
	exitOnCtrlC(err)
	if err != nil {
		return nil, err
	}

	domainConfig, _ := createDomainConfiguration(domain, provider)
	return &domainConfig, nil
}

func (d *dnsCredentialsAzure) parseCredentials() error {
	qs := []*survey.Question{
		{
			Name:      "TenantId",
			Prompt:    &survey.Input{Message: "Azure tenant ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "SubscriptionId",
			Prompt:    &survey.Input{Message: "Azure subscription ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "ClientID",
			Prompt:    &survey.Input{Message: "Azure client ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "ClientSecret",
			Prompt:    &survey.Input{Message: "Azure client secret? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	exitOnCtrlC(err)
	if err != nil {
		return err
	}

	return nil
}

func (d *dnsCredentialsOSDesignate) parseCredentials() error {
	qs := []*survey.Question{
		{
			Name:      "ApplicationCredentialID",
			Prompt:    &survey.Input{Message: "Application Credential ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "ApplicationCredentialSecret",
			Prompt:    &survey.Input{Message: "Application Credential Secret? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "AuthURL",
			Prompt:    &survey.Input{Message: "AuthURL? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required,url"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	exitOnCtrlC(err)
	if err != nil {
		return err
	}
	return nil
}

func (d *dnsCredentialsAWS53) parseCredentials() error {
	qs := []*survey.Question{
		{
			Name:      "AccessKeyID",
			Prompt:    &survey.Input{Message: "Access Key ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "SecretAccessKey",
			Prompt:    &survey.Input{Message: "Secret Access Key? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	exitOnCtrlC(err)
	if err != nil {
		return err
	}
	return nil
}

func withValidator(tag string) survey.AskOpt {
	return survey.WithValidator(common.MakeValidatorFn(tag))
}
