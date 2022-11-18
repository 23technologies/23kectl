package install

import (
	"context"
	"encoding/hex"
	"fmt"
	"html/template"
	"math/rand"
	"net"
	"os"
	"strings"

	"github.com/Masterminds/sprig/v3"
	yaml "gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// install ...

func Install(kubeconfig string, keConfiguration *KeConfig) {

	fmt.Println("Our install code here")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// keConfiguration := newKeConfig()
	completeKeConfig(keConfiguration, clientset)

	// ------ templating -----
	funcMap := sprig.FuncMap()
	funcMap["toYaml"] = func(in interface{}) string {
		result, err := yaml.Marshal(in)
		if err != nil {
			panic("Error during marshaling")
		}
		return string(result)
	}

	for i, sec := range keConfigTpl() {
		tpl := template.New(fmt.Sprint("keConfig", i)).Funcs(funcMap)
		tpl.Parse(sec)
		tpl.Execute(os.Stdout, keConfiguration)
		if err != nil {
			panic(err)
		}
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	_ = pods
}

// completeKeConfig ...
func completeKeConfig(config *KeConfig, clientset *kubernetes.Clientset) {
	if strings.TrimSpace(config.Dashboard.SessionSecret) == "" {
		tmpRand20 := make([]byte, 20)
		rand.Read(tmpRand20)
		config.Dashboard.SessionSecret = hex.EncodeToString(tmpRand20)
	}
	if strings.TrimSpace(config.Dashboard.ClientSecret) == "" {
		tmpRand20 := make([]byte, 20)
		rand.Read(tmpRand20)
		config.Dashboard.ClientSecret = hex.EncodeToString(tmpRand20)
	}
	if strings.TrimSpace(config.KubeApiServer.BasicAuthPassword) == "" {
		tmpRand20 := make([]byte, 20)
		rand.Read(tmpRand20)
		config.KubeApiServer.BasicAuthPassword = hex.EncodeToString(tmpRand20)
	}
	if strings.TrimSpace(config.ClusterIdentity) == "" {
		tmpRand5 := make([]byte, 5)
		rand.Read(tmpRand5)
		config.ClusterIdentity = "garden-cluster-" + hex.EncodeToString(tmpRand5) + "-identity"
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	config.Gardenlet.SeedPodCidr = nodes.Items[0].Spec.PodCIDR

	dummySvc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dummy",
		},
		Spec: apiv1.ServiceSpec{
			Ports:     []apiv1.ServicePort{{Name: "port", Port: 443}},
			ClusterIP: "1.1.1.1",
		},
	}
	_, dummyErr := clientset.CoreV1().Services("default").Create(context.Background(), dummySvc, metav1.CreateOptions{})

	config.Gardenlet.SeedServiceCidr = strings.SplitAfter(dummyErr.Error(), "The range of valid IPs is ")[1]

	clusterIp, ipnet, err := net.ParseCIDR(config.Gardenlet.SeedServiceCidr)

	clusterIp[len(clusterIp)-2] += 1
	clusterIp[len(clusterIp)-1] += 1
	
	if ! ipnet.Contains( clusterIp ) {
		panic("Your cluster ip is out of the service IP range")
	}
	config.Gardener.ClusterIP = clusterIp.String()

	// query for all config options we don't know
	queryConfig(config)

	// enable the provider extensions needed for a minimal setup
	config.ExtensionsConfig = make(extensionsConfig)
	config.ExtensionsConfig["provider-" + config.BaseCluster.Provider] = map[string]bool{"enabled": true}
	config.ExtensionsConfig[dnsProviderToProvider[config.DomainConfig.Provider]] = map[string]bool{"enabled": true}

}
