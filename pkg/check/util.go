package check

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var KubeClient client.Client
var KubeClientGo *kubernetes.Clientset

func init() {
	var err error

	scheme := runtime.NewScheme()
	_ = sourcev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)

	KubeClient, err = client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}

	KubeClientGo, err = kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		panic(err)
	}
}

func getMessage(conditions []v1.Condition, whereType string) string {
	for _, condition := range conditions {
		if condition.Type == whereType {
			return condition.Message
		}
	}

	return ""
}
