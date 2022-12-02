package install

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"net"
	"strings"

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

	installFlux(kubeClient, kubeconfigArgs, kubeclientOptions)

	completeKeConfig(keConfiguration, kubeClient)
	queryConfig(keConfiguration)
	// enable the provider extensions needed for a minimal setup
	keConfiguration.ExtensionsConfig = make(extensionsConfig)
	keConfiguration.ExtensionsConfig["provider-"+keConfiguration.BaseCluster.Provider] = map[string]bool{"enabled": true}
	keConfiguration.ExtensionsConfig[dnsProviderToProvider[keConfiguration.DomainConfig.Provider]] = map[string]bool{"enabled": true}

	// Generate the needed deploy keys
	fmt.Println("Generating 23ke deploy key")
	fmt.Println(`This key will need to be added by 23T to the 23KE repository.
Please contact the 23T administrators and ask them to add the key.
Depending on your relationship with 23T, 23T will come up with a pricing model for you.`)
	publicKeys23ke, err := generateDeployKey(kubeClient, "23ke-key", _23KERepoURI)
	_ = publicKeys23ke // todo use
	_panic(err)

	fmt.Println("Generating 23ke-config deploy key")
	fmt.Println(`You will need to add this key to your git remote git repository.
The key needs write access and the repository can remain empty.`)
	publicKeysConfig, err := generateDeployKey(kubeClient, "23ke-config-key", keConfiguration.GitRepo)
	_panic(err)

	create23keConfigSecret(keConfiguration, kubeClient)

	installVPACRDs(keConfiguration, kubeconfigArgs, kubeclientOptions)

	createGitRepositories(kubeClient, *keConfiguration)

	createKustomizations(kubeClient)

	err = updateConfigRepo(keConfiguration, *publicKeysConfig)
	_panic(err)

	// todo: show some kind of progress bar

	fmt.Println("")
	fmt.Println("")
	fmt.Println("Awesome. Your gardener installation should be up within 10 minutes.")
	fmt.Printf("Once it's done you can login as %s.\n", color.BlueString(keConfiguration.EmailAddress))
	fmt.Printf("Go kill some time by eagerly pressing F5 on https://dashboard.%s\n", color.BlueString(keConfiguration.DomainConfig.Domain))
}

func completeKeConfig(config *KeConfig, kubeClient client.WithWatch) {
	if strings.TrimSpace(config.Dashboard.SessionSecret) == "" {
		config.Dashboard.SessionSecret = randHex(20)
	}
	if strings.TrimSpace(config.Dashboard.ClientSecret) == "" {
		config.Dashboard.ClientSecret = randHex(20)
	}
	if strings.TrimSpace(config.KubeApiServer.BasicAuthPassword) == "" {
		config.KubeApiServer.BasicAuthPassword = randHex(20)
	}
	if strings.TrimSpace(config.ClusterIdentity) == "" {
		config.ClusterIdentity = "garden-cluster-" + randHex(5) + "-identity"
	}

	if strings.TrimSpace(config.Gardenlet.SeedPodCidr) == "" {
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
			config.Gardenlet.SeedPodCidr = ipPool.Object["spec"].(map[string]interface{})["cidr"].(string)
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
			config.Gardenlet.SeedPodCidr = ciliumConfig.Data["cluster-pool-ipv4-cidr"]
		}
	}

	if strings.TrimSpace(config.Gardenlet.SeedServiceCidr) == "" {
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
		config.Gardenlet.SeedServiceCidr = strings.SplitAfter(dummyErr.Error(), "The range of valid IPs is ")[1]
	}

	if strings.TrimSpace(config.Gardener.ClusterIP) == "" {
		clusterIp, ipnet, _ := net.ParseCIDR(config.Gardenlet.SeedServiceCidr)

		clusterIp[len(clusterIp)-2] += 1
		clusterIp[len(clusterIp)-1] += 1

		if !ipnet.Contains(clusterIp) {
			panic("Your cluster ip is out of the service IP range")
		}
		config.Gardener.ClusterIP = clusterIp.String()
	}
}
