package install

import (
	"context"
	"github.com/23technologies/23kectl/pkg/common"
	"github.com/23technologies/23kectl/pkg/logger"
	"time"

	kustomizecontrollerv1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createKustomizations(kubeClient client.Client) error {
	var err error
	log := logger.Get("createKustomizations")

	ks23keBase := kustomizecontrollerv1beta2.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kustomize.toolkit.fluxcd.io/v1beta2",
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.BASE_23KE_KS_NAME,
			Namespace: common.FLUX_NAMESPACE,
		},
		Spec: kustomizecontrollerv1beta2.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Path:  "./",
			Prune: true,
			SourceRef: kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
				Kind: "Bucket",
				Name: common.BUCKET_NAME,
			},
		},
		Status: kustomizecontrollerv1beta2.KustomizationStatus{},
	}

	err = Container.Create(context.TODO(), &ks23keBase, &client.CreateOptions{})
	if err != nil {
		log.Info("Couldn't create ks "+common.BASE_23KE_KS_NAME, "error", err)
	}

	ks23keConfig := kustomizecontrollerv1beta2.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kustomize.toolkit.fluxcd.io/v1beta2",
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.CONFIG_KS_NAME,
			Namespace: common.FLUX_NAMESPACE,
		},
		Spec: kustomizecontrollerv1beta2.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: time.Minute,
			},
			Path:  "./",
			Prune: true,
			SourceRef: kustomizecontrollerv1beta2.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: common.CONFIG_23KE_GITREPO_NAME,
			},
		},
		Status: kustomizecontrollerv1beta2.KustomizationStatus{},
	}

	err = Container.Create(context.TODO(), &ks23keConfig, &client.CreateOptions{})
	if err != nil {
		log.Info("Couldn't create ks "+common.CONFIG_KS_NAME, "error", err)
	}

	return nil
}
