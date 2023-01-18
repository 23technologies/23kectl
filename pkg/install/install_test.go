//go:build test

package install_test

import (
	"fmt"
	"github.com/23technologies/23kectl/pkg/common"
	"github.com/23technologies/23kectl/pkg/install"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"io/fs"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"math/rand"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var testConfig = map[string]any{
	"admin.email":                          "test@example.org",
	"admin.gitrepobranch":                  "test",
	"admin.gitrepourl":                     "file://./config_repo",
	"admin.password":                       "$2a$10$eWNJshWJxf24FVm4u7W1XOYiPzdSscmFgs3GVF.PYaC42DjuX1piu",
	"basecluster.hasverticalpodautoscaler": "false",
	"basecluster.nodecidr":                 "10.250.0.0/16",
	"basecluster.provider":                 "hcloud",
	"basecluster.region":                   "hel1",
	"bucket.accesskey":                     "minioadmin",
	"bucket.endpoint":                      "localhost:9000",
	"bucket.secretkey":                     "minioadmin",
	"clusteridentity":                      "garden-cluster-my-identity",
	"dashboard.clientsecret":               "bXktZGFzaGJvYXJkLWNsaWVudC1zZWNyZXQ=",
	"dashboard.sessionsecret":              "bXktZGFzaGJvYXJkLXNlc3Npb24tc2VjcmV0",
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

	"emailaddress":                    "test@example.org",
	"gardener.clusterip":              "10.0.0.1",
	"gardenlet.seednodecidr":          "10.250.0.0/16",
	"gardenlet.seedpodcidr":           "100.73.0.0/16",
	"gardenlet.seedservicecidr":       "100.88.0.0/13",
	"issuer.acme.email":               "test@example.org",
	"kubeapiserver.basicauthpassword": "my-basic-auth-password",
	"version":                         "test",
}

var bucketName = fmt.Sprint(rand.Uint32())
var configFileName = tmpFolder + "/config.yaml"

func init() {
	When("Running the `install` command", Ordered, func() {
		BeforeAll(func() {
			// var err error

			By("Using config file " + configFileName)
			viper.SetConfigFile(configFileName)

			install.TestConfig = testConfig

			//By("Using bucket " + bucketName)
			//err = s3Client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
			//Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			// _ = s3Client.RemoveBucket(context.Background(), bucketName)
		})

		XIt("should Install", func() {
			err := install.Install(testKubeConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		var kubeconfigArgs *genericclioptions.ConfigFlags
		var kubeclientOptions *runclient.Options
		var kubeClient client.WithWatch

		It("should create kube client", func(ctx SpecContext) {
			var err error
			By("asdasdasd")
			kubeconfigArgs, kubeclientOptions, kubeClient, err = install.CreateKubeClient(testKubeConfig)

			Expect(err).NotTo(HaveOccurred())
			Expect(kubeconfigArgs).NotTo(BeNil())
			Expect(kubeclientOptions).NotTo(BeNil())
			Expect(kubeClient).NotTo(BeNil())
		})

		It("should install flux", func() {
			err := install.InstallFlux(kubeconfigArgs, kubeclientOptions)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create BucketSecret", func() {
			err := install.CreateBucketSecret(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should completeKeConfig", func() {
			err := install.CompleteKeConfig(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should queryAdminConfig", func() {
			err := install.QueryAdminConfig()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should queryBaseClusterConfig", func() {
			err := install.QueryBaseClusterConfig()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should generateDeployKey", func() {
			keys, err := install.GenerateDeployKey(kubeClient, "somename", "ssh://a:b@localhost")
			Expect(err).NotTo(HaveOccurred())
			Expect(keys).NotTo(BeNil())
		})

		It("should create23keConfigSecret", func() {
			err := install.Create23keConfigSecret(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create23keBucket", func(ctx SpecContext) {
			err := install.Create23keBucket(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should createGitRepositories", func() {
			err := install.CreateGitRepositories(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should createAddonsKs", func() {
			err := install.CreateAddonsKs(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should createKustomizations", func() {
			err := install.CreateKustomizations(kubeClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create correct config files", func() {
			// Todo: Shouldn't be here
			viper.Set("extensionsConfig.provider-"+viper.GetString("baseCluster.provider")+".enabled", true)
			viper.Set("extensionsConfig."+common.DNS_PROVIDER_TO_PROVIDER[viper.GetString("domainConfig.provider")]+".enabled", true)
			viper.WriteConfig()

			tmpDir, err := os.MkdirTemp("", "23kectl-test-config-"+fmt.Sprint(time.Now().Unix())+"-*")
			Expect(err).NotTo(HaveOccurred())
			//defer os.RemoveAll(tmpDir)

			err = install.WriteConfigDir(osfs.New(tmpDir), ".")
			Expect(err).NotTo(HaveOccurred())

			actualFS := os.DirFS(tmpDir)
			expectedFS := os.DirFS("pkg/install/__fixture__/config")

			seenInExpected := map[string]bool{}

			err = fs.WalkDir(expectedFS, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				expectedContent, err := fs.ReadFile(expectedFS, path)
				Expect(err).NotTo(HaveOccurred())

				actualContent, err := fs.ReadFile(actualFS, path)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(actualContent)).To(Equal(string(expectedContent)), fmt.Sprintf("contents of %s don't match expected", path))

				seenInExpected[path] = true

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			err = fs.WalkDir(actualFS, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				Expect(seenInExpected[path]).To(BeTrue(), fmt.Sprintf("%s exists but shouldn't", path))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should updateConfigRepo", func() {
			err := install.UpdateConfigRepo(ssh.PublicKeys{})
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
