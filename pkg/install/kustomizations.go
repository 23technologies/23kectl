package install

import (
	"context"
	"time"

	"github.com/23technologies/23kectl/pkg/constants"
	kustomizecontrollerv1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createKustomizations(kubeClient client.WithWatch) {

	ks23keBase := kustomizecontrollerv1beta2.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kustomize.toolkit.fluxcd.io/v1beta2",
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.BASE_23KE_KS_NAME,
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
				Name: constants.BASE_23KE_GITREPO_NAME,
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
			Name:      constants.CONFIG_KS_NAME,
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
				Name: constants.CONFIG_23KE_GITREPO_NAME,
			},
		},
		Status: kustomizecontrollerv1beta2.KustomizationStatus{},
	}

	kubeClient.Create(context.TODO(), &ks23keConfig, &client.CreateOptions{})
}
