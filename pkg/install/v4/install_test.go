package install_test

import (
	"context"
	"fmt"
	"github.com/23technologies/23kectl/pkg/install/v4"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	fluxInstall "github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	kustomizecontrollerv1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcecontrollerv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	var configFileName = path.Join(tmpFolder, "config.yaml")
	var configRepo = path.Join(tmpFolder, "config.git")
	var configRepoUrl = "file://" + configRepo
	var configRepoBranch = "test"
	var configFixture = path.Join(cwd, "__fixture__/config")

	var testConfig = map[string]any{
		"admin.email":                          "test@example.org",
		"admin.gitrepobranch":                  configRepoBranch,
		"admin.gitrepourl":                     configRepoUrl,
		"admin.password":                       "$2a$10$eWNJshWJxf24FVm4u7W1XOYiPzdSscmFgs3GVF.PYaC42DjuX1piu",
		"basecluster.hasverticalpodautoscaler": false,
		"basecluster.nodecidr":                 "10.250.0.0/16",
		"basecluster.provider":                 "hcloud",
		"basecluster.region":                   "hel1",
		"bucket.accesskey":                     "minioadmin",
		"bucket.endpoint":                      "localhost:9000",
		"bucket.secretkey":                     "minioadmin",
		"cloudprofiles":                        []string{"hcloud", "regiocloud"},
		"clusteridentity":                      "garden-cluster-my-identity",
		"domainconfig": map[string]any{
			"credentials": map[string]string{
				"clientid":       "my-client-id",
				"clientsecret":   "my-client-secret",
				"subscriptionid": "my-subscription-id",
				"tenantid":       "my-tenantid",
			},
			"domain":   "my-domain.example.org",
			"provider": "azure-dns",
		},

		"backupconfig": map[string]any{
			"enabled":    true,
			"provider":   "azure",
			"region":     "germanywestcentral",
			"bucketname": "my-bucket-name",
			"credentials": map[string]string{
				"clientid":                "my-client-id",
				"clientsecret":            "my-client-secret",
				"subscriptionid":          "my-subscription-id",
				"tenantid":                "my-tenantid",
				"storageaccount":          "my-storage-account",
				"storageaccountaccesskey": "my-storage-account-accesskyey",
			},
		},

		"emailaddress":              "test@example.org",
		"gardener.clusterip":        "10.0.0.100",
		"gardenlet.seednodecidr":    "10.250.0.0/16",
		"gardenlet.seedpodcidr":     "100.73.0.0/16",
		"gardenlet.seedservicecidr": "100.88.0.0/13",
		"issuer.acme.email":         "test@example.org",
		"issuer.acme.server":        "example.acme.server",
		"issuer.ca":                 "my-great-ca",
		"version":                   "test",
	}

	install.Container.BlockUntilKeyCanRead = func(_ string, _ *ssh.PublicKeys, _ string) {}
	install.Container.GetSSHHostname = func(_ *url.URL) string { return "github.com" }
	install.Container.QueryConfigKey = func(configKey string, _ func() (any, error)) error {
		lc := strings.ToLower(configKey)

		result := testConfig[lc]

		if result == nil || result == "" {
			panic("key doesn't exist: " + lc)
		}

		viper.Set(configKey, result)

		return nil
	}
	install.Container.CreateFluxManifest = func() (*manifestgen.Manifest, error) {
		opts := fluxInstall.MakeDefaultOptions()
		manifest, err := fluxInstall.Generate(opts, "")
		if err != nil {
			return nil, err
		}
		return manifest, nil

	}

	When("Running the `install` command", Ordered, func() {
		var installErr error

		BeforeAll(func() {
			var err error

			viper.SetConfigFile(configFileName)
			// we need to set the version and bucket.endpoint here, as
			// it is set outside the versioned install pkg in production
			viper.Set("version", testConfig["version"])
			viper.Set("bucket.endpoint", testConfig["bucket.endpoint"])
			viper.Set("bucket.accesskey", testConfig["bucket.accesskey"])
			viper.Set("bucket.secretkey", testConfig["bucket.secretkey"])

			// set these to prevent auto-generation
			viper.Set("clusterIdentity", testConfig["clusteridentity"])
			viper.Set("cloudprofiles", testConfig["cloudprofiles"])
			viper.Set("issuer.ca", testConfig["issuer.ca"])
			viper.Set("issuer.acme.server", testConfig["issuer.acme.server"])

			_, err = git.PlainInit(configRepo, true)
			if err != nil {
				panic(err)
			}

			installErr = install.Install(testKubeConfig, false)
		})

		It("should install flux", func() {
			key := client.ObjectKey{
				Namespace: "flux-system",
				Name:      "source-controller",
			}

			deployment := appsv1.Deployment{}
			err := k8sClient.Get(context.Background(), key, &deployment)
			Expect(err).To(BeNil())
			Expect(deployment.Name).To(BeEquivalentTo("source-controller"))
		})

		It("should create BucketSecret", func() {
			key := client.ObjectKey{
				Namespace: "flux-system",
				Name:      "bucket-credentials",
			}

			secret := corev1.Secret{}
			err := k8sClient.Get(context.Background(), key, &secret)
			Expect(err).To(BeNil())
			Expect(secret.Data["accesskey"]).To(BeEquivalentTo(testConfig["bucket.accesskey"]))
			Expect(secret.Data["secretkey"]).To(BeEquivalentTo(testConfig["bucket.secretkey"]))
			Expect(secret.Type).To(BeEquivalentTo(corev1.SecretTypeOpaque))
		})

		XIt("should completeKeConfig", func() {
			// todo test detection of calico/cilium in the cluster, maybe others
		})

		XIt("should generateDeployKey", func() {
			// todo test if existing deploy key is reused
		})

		It("should create23keConfigSecret", func(ctx SpecContext) {
			expectedValues := fmt.Sprintf(`
clusterIdentity: %s
domains:
  global:
    credentials:
      clientid: %s
      clientsecret: %s
      subscriptionid: %s
      tenantid: %s
    domain: my-domain.example.org
    provider: azure-dns
issuer:
  enabled: true
  acme:
    email: test@example.org
    server: %s
  ca: |
    %s
backups:
  enabled: %t
  provider: %s
  region: %s
  bucketName: %s
  credentials:
    storageaccount: %s
    storageaccountaccesskey: %s
    clientid: %s
    clientsecret: %s
    subscriptionid: %s
    tenantid: %s
`,
				testConfig["clusteridentity"],
				testConfig["domainconfig"].(map[string]any)["credentials"].(map[string]string)["clientid"],
				testConfig["domainconfig"].(map[string]any)["credentials"].(map[string]string)["clientsecret"],
				testConfig["domainconfig"].(map[string]any)["credentials"].(map[string]string)["subscriptionid"],
				testConfig["domainconfig"].(map[string]any)["credentials"].(map[string]string)["tenantid"],
				testConfig["issuer.acme.server"],
				testConfig["issuer.ca"],
				testConfig["backupconfig"].(map[string]any)["enabled"].(bool),
				testConfig["backupconfig"].(map[string]any)["provider"].(string),
				testConfig["backupconfig"].(map[string]any)["region"].(string),
				testConfig["backupconfig"].(map[string]any)["bucketname"].(string),
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["storageaccount"],
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["storageaccountaccesskey"],
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["clientid"],
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["clientsecret"],
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["subscriptionid"],
				testConfig["backupconfig"].(map[string]any)["credentials"].(map[string]string)["tenantid"],
			)

			key := client.ObjectKey{Namespace: "flux-system", Name: "23ke-config"}
			secret := corev1.Secret{}
			err := k8sClient.Get(ctx, key, &secret)

			fmt.Println(expectedValues)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["values.yaml"]).To(MatchYAML(expectedValues))
		})

		It("should create23keBucket", func() {
			key := client.ObjectKey{
				Namespace: "flux-system",
				Name:      "23ke",
			}

			bucket := sourcecontrollerv1beta2.Bucket{}
			err := k8sClient.Get(context.Background(), key, &bucket)
			Expect(err).To(BeNil())
			Expect(bucket.Name).To(BeEquivalentTo("23ke"))
			Expect(bucket.Spec.BucketName).To(BeEquivalentTo(testConfig["version"]))
			Expect(bucket.Spec.Endpoint).To(BeEquivalentTo(testConfig["bucket.endpoint"]))
			Expect(bucket.Spec.SecretRef.Name).To(BeEquivalentTo("bucket-credentials"))
		})

		XIt("should create GitRepositories", func() {
			// todo: make this work (we're using a file:// url during the tests, but flux doesn't accept anything but http(s) and ssh)

			key := client.ObjectKey{
				Namespace: "flux-system",
				Name:      "23ke-config",
			}

			gitrepo := sourcecontrollerv1beta2.GitRepository{}
			err := k8sClient.Get(context.Background(), key, &gitrepo)
			Expect(err).To(BeNil())
			Expect(gitrepo.Name).To(BeEquivalentTo("23ke-config"))
			Expect(gitrepo.Spec.URL).To(BeEquivalentTo(testConfig["admin.gitrepourl"]))
			Expect(gitrepo.Spec.Reference).To(BeEquivalentTo(sourcecontrollerv1beta2.GitRepositoryRef{Branch: testConfig["admin.gitrepobranch"].(string)}))

			Expect(nil).To(BeNil())
		})

		It("should createKustomizations", func() {
			key := client.ObjectKey{
				Namespace: "flux-system",
				Name:      "23ke-base",
			}

			ks := kustomizecontrollerv1beta2.Kustomization{}
			err := k8sClient.Get(context.Background(), key, &ks)
			Expect(err).To(BeNil())
			Expect(ks.Name).To(BeEquivalentTo("23ke-base"))
			Expect(ks.Spec.Prune).To(BeTrue())
			Expect(ks.Spec.Path).To(BeEquivalentTo("./"))
			Expect(ks.Spec.SourceRef).To(BeEquivalentTo(
				kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
					Kind: "Bucket",
					Name: "23ke",
				}))

			key = client.ObjectKey{
				Namespace: "flux-system",
				Name:      "23ke-config",
			}
			err = k8sClient.Get(context.Background(), key, &ks)
			Expect(err).To(BeNil())
			Expect(ks.Name).To(BeEquivalentTo("23ke-config"))
			Expect(ks.Spec.Prune).To(BeTrue())
			Expect(ks.Spec.Path).To(BeEquivalentTo("./"))
			Expect(ks.Spec.SourceRef).To(BeEquivalentTo(
				kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "23ke-config",
				}))
		})

		It("should update the config repo", func() {
			// clone the config repo, remove everything, add everything from the fixture, check if worktree is clean

			configRepoClone := path.Join(tmpFolder, "config-repo-clone")
			r, err := git.PlainClone(configRepoClone, false, &git.CloneOptions{
				URL:           configRepoUrl,
				ReferenceName: plumbing.NewBranchReferenceName(configRepoBranch),
			})
			Expect(err).NotTo(HaveOccurred())

			wt, err := r.Worktree()
			Expect(err).NotTo(HaveOccurred())

			_, err = wt.Remove(".")
			Expect(err).NotTo(HaveOccurred())

			// feels terrible but is safe for testing
			err = exec.Command("sh", "-c", fmt.Sprintf("cp -r %s/* %s", configFixture, configRepoClone)).Run()
			Expect(err).NotTo(HaveOccurred())

			_, err = wt.Add(".")
			Expect(err).NotTo(HaveOccurred())

			status, err := wt.Status()
			Expect(err).NotTo(HaveOccurred())

			if !status.IsClean() {
				os.Stdout.WriteString("\n\n")

				cmd := exec.Command("git", "--no-pager", "diff", "HEAD")
				cmd.Dir = configRepoClone
				cmd.Stdout = os.Stdout
				err := cmd.Run()

				Expect(err).NotTo(HaveOccurred())
			}
			Expect(status.IsClean()).To(BeTrue())
		})

		It("shouldn't return any unexpected error", func() {
			Expect(installErr).NotTo(HaveOccurred())
		})
	})
}
