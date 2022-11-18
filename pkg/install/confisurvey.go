package install

import (
	"github.com/AlecAivazis/survey/v2"
)

func queryConfig(config *KeConfig) {

	var prompt survey.Prompt

	if config.EmailAddress == "" {
		prompt = &survey.Input{
			Message: "Please enter your email address",
		}
		survey.AskOne(prompt, &config.EmailAddress)
		config.Issuer.Acme.Email = config.EmailAddress
	}

	if config.AdminPassword == "" {
		prompt = &survey.Password{
			Message: "Please enter the administrator password to use",
		}
		survey.AskOne(prompt, &config.AdminPassword)
	}

	if config.GitRepo == "" {
		prompt = &survey.Input{
			Message: "Please enter your git repository remote, e.g. git@github.com:User/Repo.git",
		}
		survey.AskOne(prompt, &config.GitRepo)
	}
	
	if config.BaseCluster.Provider == "" {
		prompt = &survey.Select{
			Message: "Select the provider of your base cluster",
			Options: []string{"hcloud", "azure", "aws", "openstack"},
		}
		survey.AskOne(prompt, &config.BaseCluster.Provider)
	}

	if config.BaseCluster.Region == "" {
		prompt = &survey.Input{
			Message: "Please enter the region of your base cluster",
		}
		survey.AskOne(prompt, &config.BaseCluster.Region)
	}
	

	if config.BaseCluster.NodeCidr == "" {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
		}
		survey.AskOne(prompt, &config.BaseCluster.NodeCidr)
		config.Gardenlet.SeedNodeCidr = config.BaseCluster.NodeCidr
	}

	if config.DomainConfig.Domain == "" || config.DomainConfig.Provider == "" {
		config.DomainConfig = queryDomainConfig()
	}

}

func queryDomainConfig() domainConfiguration {

	var domain, provider string
	var prompt survey.Prompt

	prompt = &survey.Input{
    Message: "Please enter your domain",
	}
	survey.AskOne(prompt, &domain)

	prompt = &survey.Select{
    Message: "Define your DNS provider",
    Options: []string{"azure-dns", "openstack-designate", "aws-route53"},
	}
	survey.AskOne(prompt, &provider)

	domainConfig, _ := createDomainConfiguration(domain, provider)
	return domainConfig
	
}


func (d *dnsCredentialsAzure) parseCredentials()  {

	qs := []*survey.Question{
		{
			Name:     "TenantId",
			Prompt:   &survey.Input{Message: "Azure tenant ID?"},
			Validate: survey.Required,
		},
		{
			Name:     "SubscriptionId",
			Prompt:   &survey.Input{Message: "Azure subscription ID?"},
			Validate: survey.Required,
		},
		{
			Name:     "SecretId",
			Prompt:   &survey.Input{Message: "Azure secret ID?"},
			Validate: survey.Required,
		},
		{
			Name:     "SecretValue",
			Prompt:   &survey.Input{Message: "Azure subscription ID?"},
			Validate: survey.Required,
		},
	}

	survey.Ask(qs, d)
}
