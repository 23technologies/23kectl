package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
	"html/template"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
	"strings"
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

	file, err := os.OpenFile("out.yaml", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	for i, sec := range keConfigTpl() {
		tpl := template.New(fmt.Sprint("keConfig", i)).Funcs(funcMap)

		_, err = tpl.Parse(sec)
		if err != nil {
			panic(err)
		}

		err = tpl.Execute(file, keConfiguration)
		if err != nil {
			panic(err)
		}

		file.WriteString("\n---\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	_ = pods

	clone23KE()
	setupFlux()
	setupDeployKey()
}

func clone23KE() {
	dir := "/tmp/23ke"
	URL := "git@github.com:23technologies/23ke.git"

	// https://github.com/go-git/go-git/issues/411
	// https://github.com/golang/go/issues/29286
	_, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:      URL,
		Progress: os.Stdout,
	})

	if err != nil {
		panic(err)
	}
}

func makeTools() {
	// todo make tools
	// make -f hack/tools/tools.mk all
	// export PATH=hack/tools/bin:$PATH
}

func setupDeployKey() {
	// todo: create secret, wait for deploy key to be added
	// flux create secret git 23ke-key --url=ssh://git@github.com/23technologies/23ke
}

func setupFlux() {
	// todo: install flux
	// kubectl apply -f flux-system/gotk-components.yaml
}

// completeKeConfig ...
func completeKeConfig(config *KeConfig, clientset *kubernetes.Clientset) {
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

	if !ipnet.Contains(clusterIp) {
		panic("Your cluster ip is out of the service IP range")
	}
	config.Gardener.ClusterIP = clusterIp.String()

	// query for all config options we don't know
	queryConfig(config)

	// enable the provider extensions needed for a minimal setup
	config.ExtensionsConfig = make(extensionsConfig)
	config.ExtensionsConfig["provider-"+config.BaseCluster.Provider] = map[string]bool{"enabled": true}
	config.ExtensionsConfig[dnsProviderToProvider[config.DomainConfig.Provider]] = map[string]bool{"enabled": true}

}

func randHex(bytes int) string {
	byteArr := make([]byte, bytes)
	rand.Read(byteArr)
	return hex.EncodeToString(byteArr)
}
