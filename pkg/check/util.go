package check

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kubeClient client.Client

func init() {
	var err error

	scheme := runtime.NewScheme()
	_ = sourcev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)

	kubeClient, err = client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}
}
