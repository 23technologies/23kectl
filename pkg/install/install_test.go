//go:build test

package install_test

import (
	"context"
	"fmt"
	"math/rand"

	install "github.com/23technologies/23kectl/pkg/install"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
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
	"bucket.accesskey":                     "Q3AM3UQ867SPQQA43P2F",
	"bucket.endpoint":                      "play.min.io",
	"bucket.secretkey":                     "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
	"clusteridentity":                      "garden-cluster-my-identity",
	"dasboard.clientsecret":                "bXktZGFzaGJvYXJkLWNsaWVudC1zZWNyZXQ=",
	"dasboard.sessionsecret":               "bXktZGFzaGJvYXJkLXNlc3Npb24tc2VjcmV0",
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

	"emailaddress":                     "test@example.org",
	"gardener.clusterip":               "10.0.0.1",
	"gardenlet.seednodecidr":           "10.250.0.0/16",
	"gardenlet.seedpodcidr":            "100.73.0.0/16",
	"gardenlet.seedservicecidr":        "100.88.0.0/13",
	"issuer.acme.email":                "test@example.org",
	"kubeapiserver:.basicauthpassword": "my-basic-auth-password",
	"version":                          "test",
}

var _ = It("Should install", func() {
	configFileName := "/tmp/23kectl-config-" + fmt.Sprint(rand.Uint32()) + ".yaml"
	fmt.Println("Using " + configFileName)
	viper.SetConfigFile(configFileName)


	endpoint := testConfig["bucket.endpoint"].(string)
	accessKeyID := testConfig["bucket.accesskey"].(string)
	secretAccessKey := testConfig["bucket.secretkey"].(string)

	// Initialize minio client object.
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	Expect(err).NotTo(HaveOccurred())
	
	// Make a new bucket called mymusic.
	bucketName := fmt.Sprint(rand.Uint32())

	err = s3Client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})


	install.TestConfig = testConfig
	err = install.Install(testKubeConfig)
	Expect(err).NotTo(HaveOccurred())

	err = s3Client.RemoveBucket(context.Background(), bucketName)
	Expect(err).NotTo(HaveOccurred())

	//sourceController := client.ObjectKey{
	//	Namespace: "flux-system",
	//	Name:      "source-controller",
	//}
	//
	//testDep := new(appsv1.Deployment)
	//
	//fmt.Println(k8sClient.Get(context.Background(), sourceController, testDep))
	//fmt.Println(testDep.Name)
})
