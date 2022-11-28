package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
)

// install ...

const tmpDir = "/tmp"

func Install(kubeconfig string, keConfiguration *KeConfig) {
	makeCmd := func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command(name, arg...)
		cmd.Env = append(cmd.Environ(), "KUBECONFIG="+kubeconfig)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd
	}

	fmt.Println("Our install code here")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	_panic(err)

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	_panic(err)

	completeKeConfig(keConfiguration, clientset)

	configRepoDir := path.Join(tmpDir, "23ke-config")
	_23KERepoDir := path.Join(tmpDir, "23ke")
	os.RemoveAll(configRepoDir)
	os.RemoveAll(_23KERepoDir)

	err = updateConfigRepo(configRepoDir, keConfiguration, kubeconfig)
	_panic(err)
	os.Exit(111)
	// fmt.Printf("Cloning 23ke repo to %s\n", _23KERepoDir)
	// err = makeCmd("git", "clone", "git@github.com:23technologies/23ke.git", _23KERepoDir).Run()
	// _panic(err)
	//
	//fmt.Printf("Installing Flux\n")
	//cmd = makeCmd("kubectl", "apply", "-f", path.Join(_23KERepoDir, "flux-system", "gotk-components.yaml"))
	//err = cmd.Run()
	//_panic(err)
	//pressEnterToContinue()
	//
	//fmt.Printf("Generating 23ke deploy key\n")
	//err = makeCmd("flux", "create", "secret", "git", "23ke-key", "--url=ssh://git@github.com/23technologies/23ke").Run()
	//_panic(err)
	//pressEnterToContinue()

	//fmt.Printf("Generating 23ke-config deploy key\n")
	//err = makeCmd("flux", "create", "secret", "git", "23ke-config-key", "--url=ssh://git@github.com/j2l4e/23test").Run()
	//_panic(err)
	//pressEnterToContinue()

	fmt.Printf("Creating '23ke-config' secret\n")
	filePath := path.Join(tmpDir, "23ke-config.yaml")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		file.Close()
		panic(err)
	}
	err = getLocalTemplate().ExecuteTemplate(file, "23ke-config.yaml", keConfiguration)
	file.Close()
	_panic(err)
	err = makeCmd("kubectl", "apply", "-f", filePath).Run()
	_panic(err)

	fmt.Printf("Creating flux git source '23ke'\n")
	err = makeCmd("flux", "create", "source", "git", "23ke", "--secret-ref=23ke-key", "--url=ssh://git@github.com/23technologies/23ke", "--tag=v1.60.0", "--interval=1m").Run()
	_panic(err)

	fmt.Printf("Creating flux git source '23ke-config'\n")
	url := fmt.Sprintf("ssh://%s", strings.Replace(keConfiguration.GitRepo, ":", "/", 1))
	err = makeCmd("flux", "create", "source", "git", "23ke-config", "--secret-ref=23ke-config-key", "--url="+url, "--branch=main", "--interval=1m").Run()
	_panic(err)

	fmt.Printf("Creating kustomization '23ke-base'\n")
	err = makeCmd("flux", "create", "kustomization", "23ke-base", "--namespace=flux-system", "--source=GitRepository/23ke", `--path=./`, "--prune=false", "--interval=1m").Run()
	_panic(err)

	fmt.Printf("Creating flux git source '23ke-env'\n")
	err = makeCmd("flux", "create", "kustomization", "23ke-env", "--namespace=flux-system", "--source=GitRepository/23ke-config", `--path=./my-env`, "--prune=false", "--interval=1m").Run()
	_panic(err)

	//pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	//_panic(err)
	//
	//_ = pods
}

func updateConfigRepo(configRepoDir string, keConfig *KeConfig, kubeconfig string) error {
	makeCmd := func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command(name, arg...)
		cmd.Env = append(cmd.Environ(), "KUBECONFIG="+kubeconfig)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd
	}

	var cmd *exec.Cmd
	var err error

	fmt.Printf("Cloning config repo to %s\n", configRepoDir)
	err = makeCmd("git", "clone", keConfig.GitRepo, configRepoDir).Run()
	_panic(err)

	cmd = makeCmd("git", "rm", "-r", ".")
	cmd.Dir = configRepoDir
	err = cmd.Run()
	_panic(err)

	fmt.Printf("Writing new config to %s\n", configRepoDir)
	err = writeConfigDir(configRepoDir, keConfig)
	_panic(err)

	cmd = makeCmd("git", "add", ".")
	cmd.Dir = configRepoDir
	err = cmd.Run()
	_panic(err)

	fmt.Printf("Commiting to config repo\n")
	cmd = makeCmd("git", "commit", "--no-gpg-sign", "-m", "Config update through 23kectl") // todo prompt for commit message or let git handle it
	cmd.Dir = configRepoDir
	err = cmd.Run()
	_panic(err)

	fmt.Printf("Pushing to config repo\n")
	cmd = makeCmd("git", "push")
	cmd.Dir = configRepoDir
	err = cmd.Run()
	_panic(err)

	return nil
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

	if strings.TrimSpace(config.Gardenlet.SeedPodCidr) == "" {
		// https://github.com/gardener/gardener/blob/e31175861175410185b492b861cc90ba5491a8ee/cmd/gardenlet/app/bootstrappers/seed_config.go#L73
		// todo find proper way to find PodCIDR
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		_panic(err)
		config.Gardenlet.SeedPodCidr = nodes.Items[0].Spec.PodCIDR
	}

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

	clusterIp, ipnet, _ := net.ParseCIDR(config.Gardenlet.SeedServiceCidr)

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
