package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	k8syaml "sigs.k8s.io/yaml"
	"strings"
	"time"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"sigs.k8s.io/yaml"

	"github.com/23technologies/23kectl/pkg/utils"
	runclient "github.com/fluxcd/pkg/runtime/client"
	sourcecontrollerv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// install ...

const tmpDir = "/tmp"
const _23KERepo = "git@github.com:23technologies/23ke.git"
const _23KERepoURI = "ssh://git@github.com/23technologies/23ke.git"

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

	_23KERepoDir := path.Join(tmpDir, "23ke")
	os.RemoveAll(_23KERepoDir)

	err = updateConfigRepo(keConfiguration)
	_panic(err)

	// Install flux.
	// We just copied over github.com/fluxcd/flux2/internal/utils to 23kectl/pkg/utils
	// and use the Apply function as is
	var kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	var kubeclientOptions = new(runclient.Options)
	tmpDir, err := manifestgen.MkdirTempAbs("", *kubeconfigArgs.Namespace)
	_panic(err)

	defer os.RemoveAll(tmpDir)

	opts := install.MakeDefaultOptions()
	manifest, err := install.Generate(opts, "flux-")
	_panic(err)

	_, err = manifest.WriteFile(tmpDir)
	_panic(err)

	_, err = utils.Apply(context.Background(), kubeconfigArgs, kubeclientOptions, tmpDir, path.Join(tmpDir, manifest.Path))
	_panic(err)

	pressEnterToContinue()

	generate23KEDeployKey(clientset)

	fmt.Printf("Generating 23ke deploy key\n")
	err = makeCmd("flux", "create", "secret", "git", "23ke-key", "--url="+_23KERepoURI).Run()
	_panic(err)
	pressEnterToContinue()

	fmt.Printf("Generating 23ke-config deploy key\n")
	err = makeCmd("flux", "create", "secret", "git", "23ke-config-key", "--url=ssh://git@github.com/j2l4e/23test").Run()
	_panic(err)
	pressEnterToContinue()

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

	_23keConfigSec := apiv1.Secret{}

	tmpByte, err := os.ReadFile(file.Name())
	yaml.Unmarshal(tmpByte, &_23keConfigSec)
	clientset.CoreV1().Secrets("flux-system").Create(context.TODO(), &_23keConfigSec, metav1.CreateOptions{})

	gitrepo23ke := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke",
			Namespace: "flux-system",
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{
				Tag: keConfiguration.Version,
			},
			URL: "git@github.com:23technologies/23ke.git",
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	gitrepo23keconfig := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke-config",
			Namespace: "flux-system",
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{
				Branch: "main",
			},
			URL: keConfiguration.GitRepo,
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	kubeClient.Create(context.TODO(), &gitrepo23ke, &client.CreateOptions{})
	kubeClient.Create(context.TODO(), &gitrepo23keconfig, &client.CreateOptions{})

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

func generate23KEDeployKey(clientset *kubernetes.Clientset) error {
	var err error

	namespace := "flux-system"
	secretName := "23ke-key"
	fluxRepoSecret := corev1.Secret{}
	repourl, err := url.Parse(_23KERepoURI)
	if err != nil {
		return err
	}

	// define some options for the generation of the flux source secret
	sourceSecOpts := sourcesecret.MakeDefaultOptions()
	sourceSecOpts.PrivateKeyAlgorithm = "ed25519"
	sourceSecOpts.SSHHostname = repourl.Hostname()
	sourceSecOpts.Name = secretName

	// generate the flux source secret manifest and store it as []byte in the shootResources
	secManifest, err := sourcesecret.Generate(sourceSecOpts)

	// lastly, also deploy the flux source secret into the projectNamespace in the seed cluster
	// in order to reuse it, when other shoots are created
	err = k8syaml.Unmarshal([]byte(secManifest.Content), &fluxRepoSecret)

	_panic(err)
	fluxRepoSecret.SetNamespace(namespace)

	fmt.Println(fluxRepoSecret.StringData["identity.pub"])
	clientset.CoreV1().Secrets(namespace).Create(context.TODO(), &fluxRepoSecret, metav1.CreateOptions{})
	//a.clientGardenlet.Create(context.TODO(), &fluxRepoSecret)

	return nil
}

func updateConfigRepo(keConfig *KeConfig) error {
	var err error
	workTreeFs := memfs.New()
	fmt.Printf("Cloning config repo to memory")
	repository, err := git.Clone(memory.NewStorage(), workTreeFs, &git.CloneOptions{
		URL: keConfig.GitRepo,
	})
	_panic(err)

	worktree, err := repository.Worktree()
	_panic(err)

	_, err = worktree.Remove(".")
	_panic(err)

	fmt.Printf("Writing new config")
	err = writeConfigDir(workTreeFs, ".", keConfig)
	_panic(err)

	_, err = worktree.Add(".")
	_panic(err)

	status, err := worktree.Status()
	_panic(err)

	if status.IsClean() {
		fmt.Printf("Git reports no changes to config repo")
	} else {
		fmt.Printf("Commiting to config repo\n")
		_, err = worktree.Commit("Config update through 23kectl", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "23ke Ctl",
				Email: "23kectl@23technologies.cloud",
				When:  time.Now(),
			},
		})
		_panic(err)

		fmt.Printf("Pushing to config repo\n")
		err = repository.Push(&git.PushOptions{})
		_panic(err)
	}

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
