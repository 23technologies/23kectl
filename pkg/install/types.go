package install

type KeConfig struct {
	Version          string              `yaml:"version"`
	BaseCluster      baseClusterConfig   `yaml:"baseCluster"`
	Admin            admin               `yaml:"admin"`
	ClusterIdentity  string              `yaml:"clusterIdentity"`
	Gardener         gardenerConfig      `yaml:"gardener"`
	Gardenlet        gardenletConfig     `yaml:"gardenlet"`
	KubeApiServer    kubeApiServerConfig `yaml:"kubeApiServer"`
	Dashboard        dashboardConfig     `yaml:"dashboard"`
	Issuer           issuerConfig        `yaml:"issuer"`
	DomainConfig     domainConfiguration `yaml:"domainConfig,omitempty"`
	ExtensionsConfig extensionsConfig    `yaml:"extensions"`
}

type admin struct {
	Email         string `yaml:"email"`
	Password      string `yaml:"password"`
	GitRepoURL    string `yaml:"gitrepourl"`
	GitRepoBranch string `yaml:"gitrepobranch"`
}

type baseClusterConfig struct {
	HasVerticalPodAutoscaler *bool  `yaml:"hasVerticalPodAutoscaler"`
	Provider                 string `yaml:"provider"`
	Region                   string `yaml:"region"`
	NodeCidr                 string `yaml:"nodeCidr"`
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

type domainConfiguration struct {
	Domain      string      `yaml:"domain"`
	Provider    string      `yaml:"provider"`
	Credentials interface{} `yaml:"credentials"`
}

type dnsCredentialsAzure struct {
	TenantId       string `yaml:"tenantID"`
	SubscriptionId string `yaml:"subscriptionID"`
	ClientId       string `yaml:"clientID"`
	ClientSecret   string `yaml:"clientSecret"`
}

// https://github.com/gardener/external-dns-management/blob/master/examples/20-secret-openstack-credentials.yaml
type dnsCredentialsOSDesignate struct {
	ApplicationCredentialID     string `yaml:"OS_APPLICATION_CREDENTIAL_ID"`
	ApplicationCredentialSecret string `yaml:"OS_APPLICATION_CREDENTIAL_SECRET"`
	AuthURL                     string `yaml:"OS_AUTH_URL"`
}

// https://github.com/gardener/external-dns-management/blob/master/examples/20-secret-aws-credentials.yaml
type dnsCredentialsAWS53 struct {
	AccessKeyID     string `yaml:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `yaml:"AWS_SECRET_ACCESS_KEY"`
}

type extensionsConfig map[string]map[string]bool

const (
	PROVIDER_AWS       = "provider-aws"
	PROVIDER_AZURE     = "provider-azure"
	PROVIDER_GCP       = "provider-gcp"
	PROVIDER_OPENSTACK = "provider-openstack"
	PROVIDER_ALICLOUD  = "provider-alicloud"
)

const (
	DNS_PROVIDER_AWS_ROUTE_53        = "aws-route53"
	DNS_PROVIDER_AZURE_DNS           = "azure-dns"
	DNS_PROVIDER_AZURE_PRIVATE_DNS   = "azure-private-dns"
	DNS_PROVIDER_GOOGLE_CLOUDDNS     = "google-clouddns"
	DNS_PROVIDER_OPENSTACK_DESIGNATE = "openstack-designate"
	DNS_PROVIDER_ALICLOUD_DNS        = "alicloud-dns"
)

var dnsProviderToProvider = map[string]string{
	DNS_PROVIDER_AWS_ROUTE_53:        PROVIDER_AWS,
	DNS_PROVIDER_AZURE_DNS:           PROVIDER_AZURE,
	DNS_PROVIDER_AZURE_PRIVATE_DNS:   PROVIDER_AZURE,
	DNS_PROVIDER_GOOGLE_CLOUDDNS:     PROVIDER_GCP,
	DNS_PROVIDER_OPENSTACK_DESIGNATE: PROVIDER_OPENSTACK,
	DNS_PROVIDER_ALICLOUD_DNS:        PROVIDER_ALICLOUD,
}
