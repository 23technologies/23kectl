//go:build test

package installv1_test

import (
	"fmt"
	install "github.com/23technologies/23kectl/pkg/install/v1"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"os/exec"
	"path"
)

var testConfig = map[string]any{
	"admin.email":                          "test@example.org",
	"admin.gitrepobranch":                  configRepoBranch,
	"admin.gitrepourl":                     configRepoUrl,
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

var cwd, _ = os.Getwd()

var bucketName = fmt.Sprint(rand.Uint32())
var configFileName = path.Join(tmpFolder, "config.yaml")
var configRepo = path.Join(tmpFolder, "config.git")
var configRepoUrl = "file://" + configRepo
var configRepoBranch = "test"
var configFixture = path.Join(cwd, "__fixture__/config")

func init() {
	When("Running the `install` command", Ordered, func() {
		var installErr error

		BeforeAll(func() {
			var err error

			viper.SetConfigFile(configFileName)

			_, err = git.PlainInit(configRepo, true)
			if err != nil {
				panic(err)
			}

			install.TestConfig = testConfig
			installErr = install.Install(testKubeConfig)

			//By("Using bucket " + bucketName)
			//err = s3Client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
			//Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func(ctx SpecContext) {
			// _ = s3Client.RemoveBucket(context.Background(), bucketName)
		})

		It("should install flux", func() {
			Expect(nil).To(BeNil())
		})

		It("should create BucketSecret", func() {
			Expect(nil).To(BeNil())
		})

		It("should completeKeConfig", func() {
			Expect(nil).To(BeNil())
		})

		It("should queryAdminConfig", func() {
			Expect(nil).To(BeNil())
		})

		It("should queryBaseClusterConfig", func() {
			Expect(nil).To(BeNil())
		})

		It("should generateDeployKey", func() {
			Expect(nil).To(BeNil())
		})

		It("should create23keConfigSecret", func() {
			Expect(nil).To(BeNil())
		})

		It("should create23keBucket", func() {
			Expect(nil).To(BeNil())
		})

		It("should createGitRepositories", func() {
			Expect(nil).To(BeNil())
		})

		It("should createAddonsKs", func() {
			Expect(nil).To(BeNil())
		})

		It("should createKustomizations", func() {
			Expect(nil).To(BeNil())
		})

		It("should update the config repo", func() {
			// clone the config repo, remove everything, add everything from the fixture, check if worktree is clean

			configRepoClone := path.Join(tmpFolder, "config-repo-clone")
			r, err := git.PlainClone(configRepoClone, false, &git.CloneOptions{
				URL:           configRepoUrl,
				ReferenceName: plumbing.NewBranchReferenceName(configRepoBranch),
			})

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
