//go:build test

package install_test

import (
	"context"
	"github.com/23technologies/23kectl/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
)

var testEnv *envtest.Environment
var cancel context.CancelFunc
var k8sClient client.WithWatch

const testKubeConfig = "testKubeConfig.yaml"

var _ = BeforeSuite(func() {
	disposeLogger := logger.Init()
	defer disposeLogger()

	_, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Create the *rest.Config for creating new clients
	baseConfig := &rest.Config{
		// gotta go fast during tests -- we don't really care about overwhelming our test API server
		QPS:   1000.0,
		Burst: 2000.0,
	}

	testUserInfo := envtest.User{Name: "test", Groups: []string{"system:masters"}}
	testUser, err := testEnv.ControlPlane.AddUser(testUserInfo, baseConfig)
	kubeConfig, err := testUser.KubeConfig()
	os.WriteFile(testKubeConfig, kubeConfig, 0644)

	k8sClient, err = client.NewWithWatch(cfg, client.Options{})
	print(k8sClient)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Suite")
}
