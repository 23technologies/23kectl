//go:build test

package install_test

import (
	"fmt"
	install "github.com/23technologies/23kectl/pkg/install"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"math/rand"
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
	"bucket.accesskey":                     "my-accesskey@exmaple.org",
	"bucket.endpoint":                      "minio.my-endpoint.example.org",
	"bucket.secretkey":                     "bXktZXhhbXBsZS1zZWNyZXQta2V5",
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
	// "domainconfig.credentials.clientid":       "my-client-id",
	// "domainconfig.credentials.clientsecret":   "my-client-secret",
	// "domainconfig.credentials.subscriptionid": "my-subscription-id",
	// "domainconfig.credentials.tenantid":       "my-tenantid",
	// "domainconfig.domain":                     "my-domain.example.org",
	// "domainconfig.provider":                   "azure-dns",

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

	install.TestConfig = testConfig

	err := install.Install(testKubeConfig)
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
