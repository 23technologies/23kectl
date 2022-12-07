package install

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"

	"github.com/23technologies/23kectl/pkg/utils"

	runclient "github.com/fluxcd/pkg/runtime/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// install ...
const _23KERepoURI = "ssh://git@github.com/23technologies/23ke.git"

func Install(kubeconfig string, keConfiguration *KeConfig) {

	var kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	kubeconfigArgs.KubeConfig = &kubeconfig

	var kubeclientOptions = new(runclient.Options)
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		err := fmt.Errorf("error during creation of kubeclient %s", err)
		fmt.Println(err.Error())
		return
	}

	installFlux(kubeClient, kubeconfigArgs, kubeclientOptions)

	completeKeConfig(kubeClient)
	UnmarshalKeConfig(keConfiguration)
	viper.WriteConfig()

	queryAdminConfig()
	queryBaseClusterConfig()

	// Generate the needed deploy keys
	fmt.Println("Generating 23ke deploy key")
	fmt.Println(`This key will need to be added by 23T to the 23KE repository.
Please contact the 23T administrators and ask them to add the key.
Depending on your relationship with 23T, 23T will come up with a pricing model for you.`)
	publicKeys23ke, err := generateDeployKey(kubeClient, "23ke-key", _23KERepoURI)
	_panic(err)

	fmt.Println("Generating 23ke-config deploy key")
	fmt.Println(`You will need to add this key to your git remote git repository.`)
	printWarn("This key needs write access!")
	publicKeysConfig, err := generateDeployKey(kubeClient, "23ke-config-key", viper.GetString("admin.gitrepourl"))
	_panic(err)

	create23keConfigSecret(kubeClient)

	installVPACRDs(kubeconfigArgs, kubeclientOptions)

	createGitRepositories(kubeClient, publicKeys23ke)

	createKustomizations(kubeClient)

	// enable the provider extensions needed for a minimal setup
	viper.Set("extensionsConfig.provider-" + viper.GetString("baseCluster.provider") + ".enabled", true)
	viper.Set("extensionsConfig." + dnsProviderToProvider[viper.GetString("domainConfig.provider")] + ".enabled", true)
	viper.WriteConfig()
	viper.Unmarshal(keConfiguration)

	err = updateConfigRepo(*publicKeysConfig)
	_panic(err)

	// todo: show some kind of progress bar

	fmt.Println("")
	fmt.Println("")
	fmt.Println("Awesome. Your gardener installation should be up within 10 minutes.")
	fmt.Printf("Once it's done you can login as %s.\n", color.BlueString(keConfiguration.Admin.Email))
	fmt.Printf("Go kill some time by eagerly pressing F5 on https://dashboard.%s\n", color.BlueString(keConfiguration.DomainConfig.Domain))
}

func completeKeConfig(kubeClient client.WithWatch) {

	viper.SetDefault("dashboard.sessionSecret", randHex(20))
	viper.SetDefault("dashboard.clientSecret", randHex(20))
	viper.SetDefault("kubeApiServer.basicAuthPassword", randHex(20))
	viper.SetDefault("clusterIdentity", "garden-cluster-" + randHex(5) + "-identity")

	if !viper.IsSet("gardenlet.seedPodCidr") {
		// We assume that either calico or cilium are used as CNI
		// Therefore, we search for an ippool with name "default-ipv4-ippool" for the calico case.
		// In the cilium case, we search for the configmap "cilium-config" in the kube-system namespace
		// If none of these are found, we throw an error.
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
			viper.Set("gardenlet.SeedPodCidr",ipPool.Object["spec"].(map[string]interface{})["cidr"].(string))
		} else {

			ciliumConfig := corev1.ConfigMap{}
			err = kubeClient.Get(context.Background(), client.ObjectKey{
				Namespace: "kube-system",
				Name:      "cilium-config",
			}, &ciliumConfig)
			if err != nil {
				fmt.Println("I could not find the cilium-config configmap in your kube-systemnamespace")
				panic(err)
			}
			viper.Set("gardenlet.seedPodCidr", ciliumConfig.Data["cluster-pool-ipv4-cidr"])
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
		viper.Set("gardenlet.seedServiceCidr",strings.SplitAfter(dummyErr.Error(), "The range of valid IPs is ")[1])
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
}
