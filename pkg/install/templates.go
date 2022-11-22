package install

import (
	"embed"
	"fmt"
	"gopkg.in/yaml.v3"
)

func newDomainConfigAzure(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAzure
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    "azure-dns",
		Credentials: &dnsCredentials,
	}
}

func (d domainConfiguration) marshal() string {

	result, err := yaml.Marshal(d)
	if err != nil {
		panic("Error during marshaling")
	}

	return string(result)
}

func createDomainConfiguration(domain string, dnsProvider string) (domainConfiguration, error) {

	switch dnsProvider {
	case "azure-dns":
		return newDomainConfigAzure(domain), nil
	}

	return domainConfiguration{}, fmt.Errorf("input invalid for domain configuration")
}

func keConfigTpl() []string {
	return []string{
		readTemplate("23ke-config.yaml"),
		readTemplate("gardener-values.yaml"),
		readTemplate("gardenlet-values.yaml"),
		readTemplate("extensions-values.yaml"),
		readTemplate("dashboard-values.yaml"),
		readTemplate("identity-values.yaml"),
		readTemplate("cloudprofiles-values.yaml"),
	}
}

//go:embed templates/*.yaml
var templates embed.FS

func readTemplate(name string) string {
	bytes, err := templates.ReadFile("templates/" + name)

	if err != nil {
		panic(err)
	}

	return string(bytes)
}
