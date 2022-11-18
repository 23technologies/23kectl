package install

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type KeConfig struct {
	Version          string              `yaml:"version"`
	GitRepo          string              `yaml:"gitrepo"`
	BaseCluster      baseClusterConfig   `yaml:"baseCluster"`
	EmailAddress     string              `yaml:"emailAddress"`
	AdminPassword    string              `yaml:"adminPassword"`
	ClusterIdentity  string              `yaml:"clusterIdentity"`
	Gardener         gardenerConfig      `yaml:"gardener"`
	Gardenlet        gardenletConfig     `yaml:"gardenlet"`
	KubeApiServer    kubeApiServerConfig `yaml:"kubeApiServer"`
	Dashboard        dashboardConfig     `yaml:"dashboard"`
	Issuer           issuerConfig        `yaml:"issuer"`
	DomainConfig     domainConfiguration `yaml:"domainConfig,omitempty"`
	ExtensionsConfig extensionsConfig    `yaml:"extensions"`
}

type baseClusterConfig struct {
	Provider string `yaml:"provider"`
	Region   string `yaml:"region"`
	NodeCidr string `yaml:"nodeCidr"`
}

type gardenerConfig struct {
	ClusterIP string `yaml:"clusterIP"`
}

type gardenletConfig struct {
	SeedNodeCidr    string `yaml:"seedNodeCidr"`
	SeedPodCidr     string `yaml:"seedPodCidr"`
	SeedServiceCidr string `yaml:"seedServiceCidr"`
}

type dashboardConfig struct {
	ClientSecret  string `yaml:"clientSecret"`
	SessionSecret string `yaml:"sessionSecret"`
}

type kubeApiServerConfig struct {
	BasicAuthPassword string `yaml:"basicAuthPassword"`
}

type issuerConfig struct {
	Acme acmeConfig `yaml:"acme"`
	Ca   string     `yaml:"ca,omitempty"`
}

type acmeConfig struct {
	Email  string `yaml:"email"`
	Server string `yaml:"server,omitempty"`
}

type dnsCredentials interface {
	parseCredentials()
}

type domainConfiguration struct {
	Domain      string         `yaml:"domain"`
	Provider    string         `yaml:"provider"`
	Credentials dnsCredentials `yaml:"-"`
}

func (s *domainConfiguration) UnmarshalYAML(n *yaml.Node) error {

	type S domainConfiguration
	type T struct {
		*S          `yaml:",inline"`
		Credentials yaml.Node `yaml:"credentials"`
	}

	obj := &T{S: (*S)(s)}
	if err := n.Decode(obj); err != nil {
		return err
	}
	switch s.Provider {
	case "azure-dns":
		s.Credentials = new(dnsCredentialsAzure)
	default:
		panic("provider unknown")
	}

	fmt.Println("debug")
	return obj.Credentials.Decode(s.Credentials)

}

// func (s *domainConfiguration) MarshalYAML() (interface{}, error) {

// 	type S domainConfiguration
// 	type T struct {
// 		*S          `yaml:",inline"`
// 		Credentials yaml.Node `yaml:"credentials"`
// 	}

// }

type dnsCredentialsAzure struct {
	TenantId       string `yaml:"tenantId"`
	SubscriptionId string `yaml:"subscriptionId"`
	SecretId       string `yaml:"secretId"`
	SecretValue    string `yaml:"secretValue"`
}

type extensionsConfig map[string]map[string]bool

var dnsProviderToProvider = map[string]string{
	"aws-route53":         "provider-aws",
	"azure-dns":           "provider-azure",
	"azure-private-dns":   "provider-azure",
	"google-clouddns":     "provider-gcp",
	"openstack-designate": "provider-openstack",
	"alicloud-dns":        "provider-alicloud",
}
