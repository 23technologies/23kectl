package install

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/23technologies/23kectl/pkg/logger"
	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	utils "github.com/23technologies/23kectl/pkg/flux_utils"

	runclient "github.com/fluxcd/pkg/runtime/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// install ...

func Install(kubeconfig string, keConfiguration *KeConfig) error {
	log := logger.Get("Install")

	var err error
	var kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	kubeconfigArgs.KubeConfig = &kubeconfig

	var kubeclientOptions = new(runclient.Options)
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		log.Error(err, "Couldn't create kubeclient")
		return err
	}

	err = installFlux(kubeClient, kubeconfigArgs, kubeclientOptions)
	if err != nil {
		log.Error(err, "Couldn't install flux")
		return err
	}

	err = createBucketSecret(kubeClient)
	if err != nil {
		return err
	}

	err = completeKeConfig(kubeClient)
	if err != nil {
		return err
	}

	err = UnmarshalKeConfig(keConfiguration)
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

	fmt.Println("Generating 23ke-config deploy key")
	fmt.Println(`You will need to add this key to your git remote git repository.`)
	common.PrintWarn("This key needs write access!")
	publicKeysConfig, err := generateDeployKey(kubeClient, common.CONFIG_23KE_GITREPO_KEY, viper.GetString("admin.gitrepourl"))
	if err != nil {
		return err
	}

	err = create23keConfigSecret(kubeClient)
	if err != nil {
		return err
	}

	err = create23keBucket(kubeClient)
	if err != nil {
		return err
	}

	err = createGitRepositories(kubeClient)
	if err != nil {
		return err
	}

	err = createAddonsKs(kubeClient)
	if err != nil {
		return err
	}

	err = createKustomizations(kubeClient)
	if err != nil {
		return err
	}

	// enable the provider extensions needed for a minimal setup
	viper.Set("extensionsConfig.provider-"+viper.GetString("baseCluster.provider")+".enabled", true)
	viper.Set("extensionsConfig."+common.DNS_PROVIDER_TO_PROVIDER[viper.GetString("domainConfig.provider")]+".enabled", true)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}
	err = viper.Unmarshal(keConfiguration)
	if err != nil {
		return err
	}

	err = updateConfigRepo(*publicKeysConfig)
	if err != nil {
		return err
	}

	// todo: show some kind of progress bar

	fmt.Println("")
	fmt.Println("")
	fmt.Println("Awesome. Your gardener installation should be up within 10 minutes.")
	fmt.Printf("Once it's done you can login as %s.\n", color.BlueString(keConfiguration.Admin.Email))
	fmt.Printf("Go kill some time by eagerly pressing F5 on https://dashboard.%s\n", color.BlueString(keConfiguration.DomainConfig.Domain))
	return nil
}

func completeKeConfig(kubeClient client.WithWatch) error {

	viper.SetDefault("dashboard.sessionSecret", common.RandHex(20))
	viper.SetDefault("dashboard.clientSecret", common.RandHex(20))
	viper.SetDefault("kubeApiServer.basicAuthPassword", common.RandHex(20))
	viper.SetDefault("clusterIdentity", "garden-cluster-"+common.RandHex(5)+"-identity")

	if !viper.IsSet("gardenlet.seedPodCidr") {
		// We assume that either calico or cilium are used as CNI
		// Therefore, we search for an ippool with name "default-ipv4-ippool" for the calico case.
		// In the cilium case, we search for the configmap "cilium-config" in the kube-system namespace
		// If none of these are found, we ask the user for input
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
			viper.Set("gardenlet.SeedPodCidr", ipPool.Object["spec"].(map[string]interface{})["cidr"].(string))
		} else {

			ciliumConfig := corev1.ConfigMap{}
			err = kubeClient.Get(context.Background(), client.ObjectKey{
				Namespace: "kube-system",
				Name:      "cilium-config",
			}, &ciliumConfig)
			if err != nil {
				// we did not find calico or cilium related config
				// let's prompt for the Pod CIDR
				prompt := &survey.Input{
					Message: "Please enter the pod CIDR of your base cluster in the form: x.x.x.x/y",
				}
				var queryResult string
				err := survey.AskOne(prompt, &queryResult, withValidator("required,cidr"))
				exitOnCtrlC(err)
				if err != nil {
					return err
				}
				viper.Set("gardenlet.seedPodCidr", queryResult)
				viper.WriteConfig()

			} else {
				viper.Set("gardenlet.seedPodCidr", ciliumConfig.Data["cluster-pool-ipv4-cidr"])
			}
		}
	}

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
		dummyErr := kubeClient.Create(context.Background(), dummySvc)
		viper.Set("gardenlet.seedServiceCidr", strings.SplitAfter(dummyErr.Error(), "The range of valid IPs is ")[1])
	}

	if !viper.IsSet("gardener.clusterIP") {
		clusterIp, ipnet, _ := net.ParseCIDR(viper.GetString("gardenlet.seedServiceCidr"))

		clusterIp[len(clusterIp)-2] += 1
		clusterIp[len(clusterIp)-1] += 1

		if !ipnet.Contains(clusterIp) {
			panic("Your cluster ip is out of the service IP range")
		}
		viper.Set("gardener.clusterIP", clusterIp.String())
	}

	return nil
}

func getKeConfig() (*KeConfig, error) {
	keConfig := new(KeConfig)
	err := UnmarshalKeConfig(keConfig)
	if err != nil {
		return nil, nil
	}

	return keConfig, nil
}

// unmarshalKeConfig ...
func UnmarshalKeConfig(config *KeConfig) error {

	err := viper.Unmarshal(config)
	if err != nil {
		return err
	}

	_, ok := (config.DomainConfig.Credentials).(map[string]interface{})
	if ok {
		var creds interface{}
		switch config.DomainConfig.Provider {
		case common.DNS_PROVIDER_AZURE_DNS:
			creds = dnsCredentialsAzure{}
		case common.DNS_PROVIDER_OPENSTACK_DESIGNATE:
			creds = dnsCredentialsOSDesignate{}
		case common.DNS_PROVIDER_AWS_ROUTE_53:
			creds = dnsCredentialsAWS53{}
		}
		err = mapstructure.Decode(config.DomainConfig.Credentials, &creds)
		if err != nil {
			return err
		}
		config.DomainConfig.Credentials = creds
	}
	return nil
}
