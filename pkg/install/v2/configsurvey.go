package install

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/23technologies/23kectl/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/23technologies/23kectl/pkg/logger"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func queryConfig(kubeClient client.Client) error {

	log := logger.Get("queryConfig")

	err := completeKeConfig(kubeClient)
	if err != nil {
		return err
	}

	err = viper.WriteConfig()
	if err != nil {
		log.Info("Viper couldn't write config file", "error", err)
	}

	err = queryAdminConfig()
	if err != nil {
		return err
	}

	err = queryBaseClusterConfig()
	if err != nil {
		return err
	}

	if !viper.IsSet("cloudprofiles") {
		viper.Set("cloudprofiles", []string{"alicloud", "aws", "azure", "gcp", "hcloud", "regiocloud", "wavestack"})
	}

	// enable the provider extensions needed for a minimal setup
	viper.Set("extensionsConfig.provider-"+viper.GetString("baseCluster.provider")+".enabled", true)
	viper.Set("extensionsConfig."+common.DNS_PROVIDER_TO_PROVIDER[viper.GetString("domainConfig.provider")]+".enabled", true)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	return nil

}

func completeKeConfig(kubeClient client.Client) error {
	viper.SetDefault("dashboard.sessionSecret", common.RandHex(20))
	viper.SetDefault("dashboard.clientSecret", common.RandHex(20))
	viper.SetDefault("kubeApiServer.basicAuthPassword", common.RandHex(20))
	viper.SetDefault("clusterIdentity", "garden-cluster-"+common.RandHex(5)+"-identity")

	Container.QueryConfigKey("gardenlet.seedPodCidr", func() (any, error) {
		// If either calico or cilium are used as CNI we can pull the needed info from the cluster
		// Otherwise prompt the user

		// CALICO
		// If calico's installed there's an ippool with name "default-ipv4-ippool".
		ipPool := unstructured.Unstructured{}
		gvk := schema.GroupVersionKind{
			Group:   "crd.projectcalico.org",
			Version: "v1",
			Kind:    "ippool",
		}
		ipPool.SetGroupVersionKind(gvk)
		err := kubeClient.Get(context.Background(), client.ObjectKey{
			Namespace: "",
			Name:      "default-ipv4-ippool",
		}, &ipPool)
		if err == nil {
			return ipPool.Object["spec"].(map[string]interface{})["cidr"].(string), nil
		}

		// CILIUM
		// cilium uses a configmap "cilium-config" in the kube-system namespace
		ciliumConfig := corev1.ConfigMap{}
		err = kubeClient.Get(context.Background(), client.ObjectKey{
			Namespace: "kube-system",
			Name:      "cilium-config",
		}, &ciliumConfig)
		if err == nil {
			return ciliumConfig.Data["cluster-pool-ipv4-cidr"], nil
		}

		// UNKNOWN
		// let's prompt for the Pod CIDR
		prompt := &survey.Input{
			Message: "Please enter the pod CIDR of your base cluster in the form: x.x.x.x/y",
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required,cidr"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	if !viper.IsSet("gardenlet.seedServiceCidr") {
		dummySvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dummy",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{Name: "port", Port: 443}},
				ClusterIP: "1.1.1.1",
			},
		}
		dummyErr := Container.Create(context.Background(), dummySvc)
		viper.Set("gardenlet.seedServiceCidr", strings.SplitAfter(dummyErr.Error(), "The range of valid IPs is ")[1])
	}

	if !viper.IsSet("gardener.clusterIP") {
		seedServiceCidr := viper.GetString("gardenlet.seedServiceCidr")
		clusterIp, ipnet, _ := net.ParseCIDR(seedServiceCidr)

		// clusterIp[len(clusterIp)-2] += 1
		clusterIp[len(clusterIp)-1] += 100

		if !ipnet.Contains(clusterIp) {
			panic(fmt.Sprintf("Your cluster ip (%s) is out of the service IP range: %s", clusterIp, ipnet.String()))
		}
		viper.Set("gardener.clusterIP", clusterIp.String())
	}

	return nil
}

// queryAdminConfig ...
func queryAdminConfig() error {
	var err error
	var prompt survey.Prompt

	Container.QueryConfigKey("admin.email", func() (any, error) {
		prompt = &survey.Input{
			Message: `Please enter your email address.
This will be the email address to use, when you want to login to the Gardener dashboard.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required,email"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return "", err
		}

		return queryResult, nil
	})

	Container.QueryConfigKey("admin.password", func() (any, error) {
		prompt = &survey.Password{
			Message: `Please enter the administrator password to use.
This will be the password to use, when you login to the Gardener dashboard.`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return "", err
		}

		hash, err := bcrypt.GenerateFromPassword(([]byte)(queryResult), 10)
		common.ExitOnCtrlC(err)
		if err != nil {
			return "", err
		}

		return string(hash), nil
	})

	Container.QueryConfigKey("admin.gitrepourl", func() (any, error) {
		prompt = &survey.Input{
			Message: "Please enter an ssh git remote in URL form. e.g. ssh://git@github.com/User/Repo.git",
			Help: `
Configuration files are to be stored in this repo.
Flux will monitor these files to pick up configuration changes.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required,url,startswith=ssh://"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	Container.QueryConfigKey("admin.gitrepobranch", func() (any, error) {
		prompt = &survey.Input{
			Message: "Please enter the git branch to use. Will be created if it doesn't exist.",
			Default: "main",
			Help: `
Can be any branch name you want.
You can store configuration files for multiple gardeners (e.g. prod, staging, dev) on the same repo by choosing unique branch names for them.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	Container.QueryConfigKey("issuer.acme.email", func() (any, error) {
		prompt := &survey.Input{
			Message: "Please enter your email address for acme certificate generation",
			Default: viper.GetString("admin.email"),
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, common.WithValidator("required,email"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}

		return queryResult, nil
	})

	Container.QueryConfigKey("domainConfig", func() (any, error) {
		domainConfig, err := queryDomainConfig()
		if err != nil {
			return nil, err
		}
		return domainConfig, nil
	})

	Container.QueryConfigKey("backupConfig", func() (any, error) {
		backupConfig, err := queryBackupConfig()
		if err != nil {
			return nil, err
		}
		return backupConfig, nil
	})

	return nil
}

func queryBaseClusterConfig() error {
	var err error
	var prompt survey.Prompt

	// todo explain to user. what's this for?
	Container.QueryConfigKey("baseCluster.provider", func() (any, error) {
		prompt = &survey.Select{
			Message: "Select the provider of your base cluster",
			Options: []string{"hcloud", "azure", "aws", "openstack"},
			Help: `
Currently, this tools supports the listed providers for base clusters.
If you feel like this list in incomplete, contact the 23T support.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	Container.QueryConfigKey("baseCluster.Region", func() (any, error) {
		prompt = &survey.Input{
			Message: "Please enter the region of your base cluster",
			Help: `
This is the region your base cluster runs in.
Generally this is dependent on the provider of your base cluster.
For clusters hosted on Azure, this could be e.g. germanywestcentral or westeurope.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	// todo explain to user. document where to find it on supported providers
	Container.QueryConfigKey("baseCluster.nodeCidr", func() (any, error) {
		prompt = &survey.Input{
			Message: "Please enter the node CIDR of your base cluster in the form: x.x.x.x/y",
			Help: `
Gardener will check whether the nodes' ip addresses of your base cluster lie in the specified network.
Therefore, the node CIDR should match a network that comprises all ip addresses of your nodes.
`,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, common.WithValidator("required,cidr"))
		common.ExitOnCtrlC(err)
		if err != nil {
			return nil, err
		}
		return queryResult, nil
	})

	Container.QueryConfigKey("gardenlet.seedNodeCidr", func() (any, error) {
		return viper.GetString("baseCluster.nodeCidr"), nil
	})

	Container.QueryConfigKey("baseCluster.hasVerticalPodAutoscaler", func() (any, error) {
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
		common.ExitOnCtrlC(err)
		if err != nil {
			return false, err
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
		return hasVerticalPodAutoscaler, nil
	})

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
	err = survey.AskOne(prompt, &domain, common.WithValidator("required,fqdn"))
	common.ExitOnCtrlC(err)
	if err != nil {
		return nil, err
	}

	prompt = &survey.Select{
		Message: "Define your DNS provider",
		Options: []string{common.DNS_PROVIDER_AZURE_DNS, common.DNS_PROVIDER_OPENSTACK_DESIGNATE, common.DNS_PROVIDER_AWS_ROUTE_53},
	}
	err = survey.AskOne(prompt, &provider, common.WithValidator("required"))
	common.ExitOnCtrlC(err)
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
	common.ExitOnCtrlC(err)
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
	common.ExitOnCtrlC(err)
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
	common.ExitOnCtrlC(err)
	if err != nil {
		return err
	}
	return nil
}

func queryBackupConfig() (*backupConfiguration, error) {
	var err error
	var region, provider, bucketName, configureBackups string
	var enabled bool
	var prompt survey.Prompt

	prompt = &survey.Select{
		Message: `Please tell me whether you want to configure the backup functionality.`,
		Options: []string{"yes", "no"},
	}

	err = survey.AskOne(prompt, &configureBackups, common.WithValidator("required"))
	if configureBackups == "yes" {
		enabled = true
	} else {
		enabled = false
	}
	common.ExitOnCtrlC(err)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &backupConfiguration{
			Enabled: false,
		}, nil
	}

	prompt = &survey.Select{
		Message: "Define your backup provider",
		Options: []string{"azure"},
	}
	err = survey.AskOne(prompt, &provider, common.WithValidator("required"))
	common.ExitOnCtrlC(err)
	if err != nil {
		return nil, err
	}

	prompt = &survey.Input{Message: `Define the region in which your backup bucket is hosted.`}
	err = survey.AskOne(prompt, &region, common.WithValidator("required"))
	common.ExitOnCtrlC(err)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}

	prompt = &survey.Input{Message: `Define the name for the backup bucket.`}
	err = survey.AskOne(prompt, &bucketName, common.WithValidator("required"))
	common.ExitOnCtrlC(err)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}

	backupConfig, _ := createBackupConfiguration(provider)
	backupConfig.Region = region
	backupConfig.BucketName = bucketName
	return &backupConfig, nil
}

func (d *backupCredentialsAzure) parseCredentials() error {
	qs := []*survey.Question{
		{
			Name:      "TenantID",
			Prompt:    &survey.Input{Message: "Azure tenant ID? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "SubscriptionID",
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
		{
			Name:      "StorageAccount",
			Prompt:    &survey.Input{Message: "Azure storage account? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
		{
			Name:      "StorageAccountAccessKey",
			Prompt:    &survey.Input{Message: "Azure storage account accesskey? (plain or base64)"},
			Validate:  common.MakeValidatorFn("required"),
			Transform: survey.TransformString(common.CoerceBase64String),
		},
	}

	err := survey.Ask(qs, d)
	common.ExitOnCtrlC(err)
	if err != nil {
		return err
	}

	return nil
}
