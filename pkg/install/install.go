package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/23technologies/23kectl/pkg/utils"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/pkg/apis/meta"

	kustomizecontrollerv1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcecontrollerv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	runclient "github.com/fluxcd/pkg/runtime/client"
)

// install ...
const _23KERepoURI = "ssh://git@github.com/23technologies/23ke.git"

func Install(kubeconfig string, keConfiguration *KeConfig) {

	var kubeconfigArgs = genericclioptions.NewConfigFlags(false)
	kubeconfigArgs.KubeConfig = &kubeconfig

	var kubeclientOptions = new(runclient.Options)
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)

	completeKeConfig(keConfiguration, kubeClient)

	installFlux(kubeClient, kubeconfigArgs, kubeclientOptions)

	// Generate the needed deploy keys
	fmt.Println("Generating 23ke deploy key")
	fmt.Println(`This key will need to be added by 23T to the 23KE repository.
Please contact the 23T administrators and ask them to add the key.
Depending on your relationship with 23T, 23T will come up with a pricing model for you.`)
	err = generate23KEDeployKey(kubeClient, "23ke-key", _23KERepoURI)
	_panic(err)
	pressEnterToContinue()

	fmt.Println("Generating 23ke-config deploy key")
	fmt.Println(`You will need to add this key to your git remote git repository.
The key needs write access and the repository can remain empty.`)
	err = generate23KEDeployKey(kubeClient, "23ke-config-key", keConfiguration.GitRepo)
	_panic(err)
	pressEnterToContinue()

	// Create the 23ke-config secret
	fmt.Println("Creating '23ke-config' secret")
	filePath := path.Join("/tmp", "23ke-config.yaml")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		file.Close()
		panic(err)
	}
	err = getLocalTemplate().ExecuteTemplate(file, "23ke-config.yaml", keConfiguration)
	file.Close()
	_panic(err)

	_23keConfigSec := corev1.Secret{}
	tmpByte, err := os.ReadFile(file.Name())
	k8syaml.Unmarshal(tmpByte, &_23keConfigSec)
	kubeClient.Create(context.Background(), &_23keConfigSec)

	// create the gitrepository resources in the cluster
	createGitRepositories(kubeClient, *keConfiguration)

	// create the kustomization resources in the cluster
	createKustomizations(kubeClient)

	// finally update the config repository with the current configuration
	sec := corev1.Secret{}
	kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "flux-system",
		Name:      "23ke-config-key",
	}, &sec)
	publicKeys, err := ssh.NewPublicKeys("git", sec.Data["identity"], "")

	err = updateConfigRepo(keConfiguration, *publicKeys)
	_panic(err)
}

func generate23KEDeployKey(kubeClient client.WithWatch, secretName string, repoUrl string) error {
	namespace := "flux-system"

	// todo check if exists
	sec := corev1.Secret{}
	exists := false
	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}, &sec)
	if err == nil {
		exists = true
	}
	if exists {
		// todo display if exists
		if err != nil {
			return err
		}
		fmt.Println(string(sec.Data["identity.pub"]))
		return nil
	} else {
		fluxRepoSecret := corev1.Secret{}
		repourl, err := url.Parse(repoUrl)
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
		kubeClient.Create(context.Background(), &fluxRepoSecret)
	}

	return nil
}

// createGitRepositories ...
func createGitRepositories(kubeClient client.WithWatch, keConfiguration KeConfig) {

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
			URL:       _23KERepoURI,
			SecretRef: &meta.LocalObjectReference{Name: "23ke-key"},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Tag: keConfiguration.Version},
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	kubeClient.Create(context.TODO(), &gitrepo23ke, &client.CreateOptions{})

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
			URL:       keConfiguration.GitRepo,
			SecretRef: &meta.LocalObjectReference{Name: "23ke-config-key"},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Branch: "master"},
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	kubeClient.Create(context.TODO(), &gitrepo23keconfig, &client.CreateOptions{})
}

// createKustomizations ...
func createKustomizations(kubeClient client.WithWatch) {

	ks23keBase := kustomizecontrollerv1beta2.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kustomize.toolkit.fluxcd.io/v1beta2",
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke-base",
			Namespace: "flux-system",
		},
		Spec: kustomizecontrollerv1beta2.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Path:  "./",
			Prune: true,
			SourceRef: kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: "23ke",
			},
		},
		Status: kustomizecontrollerv1beta2.KustomizationStatus{},
	}

	kubeClient.Create(context.TODO(), &ks23keBase, &client.CreateOptions{})

	ks23keConfig := kustomizecontrollerv1beta2.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kustomize.toolkit.fluxcd.io/v1beta2",
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke-config",
			Namespace: "flux-system",
		},
		Spec: kustomizecontrollerv1beta2.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Path:  "./",
			Prune: true,
			SourceRef: kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: "23ke-config",
			},
		},
		Status: kustomizecontrollerv1beta2.KustomizationStatus{},
	}

	kubeClient.Create(context.TODO(), &ks23keConfig, &client.CreateOptions{})
}

func updateConfigRepo(keConfig *KeConfig, publicKeys ssh.PublicKeys) error {
	var err error
	workTreeFs := memfs.New()
	fmt.Printf("Cloning config repo to memory")
	repository, err := git.Clone(memory.NewStorage(), workTreeFs, &git.CloneOptions{
		Auth: &publicKeys,
		URL:  keConfig.GitRepo,
	})
	// _panic(err)

	worktree, err := repository.Worktree()
	// _panic(err)

	_, err = worktree.Remove(".")
	// _panic(err)

	fmt.Printf("Writing new config")
	err = writeConfigDir(workTreeFs, ".", keConfig)
	// _panic(err)

	_, err = worktree.Add(".")
	// _panic(err)

	status, err := worktree.Status()
	// _panic(err)

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
		// _panic(err)

		fmt.Printf("Pushing to config repo\n")
		err = repository.Push(&git.PushOptions{
			Auth: &publicKeys,
		})
		_panic(err)
	}

	return nil
}

// completeKeConfig ...
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
				Name: "dummy",
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

// installFlux ...
func installFlux(kubeClient client.WithWatch, kubeconfigArgs *genericclioptions.ConfigFlags, kubeclientOptions *runclient.Options)  {
	// Install flux.
	// We just copied over github.com/fluxcd/flux2/internal/utils to 23kectl/pkg/utils
	// and use the Apply function as is
	tmpDir, err := manifestgen.MkdirTempAbs("", *kubeconfigArgs.Namespace)
	_panic(err)

	defer os.RemoveAll(tmpDir)

	opts := install.MakeDefaultOptions()
	manifest, err := install.Generate(opts, "")
	_panic(err)

	_, err = manifest.WriteFile(tmpDir)
	_panic(err)

	_, err = utils.Apply(context.Background(), kubeconfigArgs, kubeclientOptions, tmpDir, path.Join(tmpDir, manifest.Path))
	_panic(err)
}
