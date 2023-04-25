package check

import (
	"context"
	"github.com/fluxcd/kustomize-controller/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KustomizationCheck struct {
	Name      string
	Namespace string
}

func (d *KustomizationCheck) GetName() string {
	return d.Name
}

func (d *KustomizationCheck) Run() *Result {
	result := &Result{}

	ks := &v1beta2.Kustomization{}

	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}, ks)

	if err != nil {
		result.IsError = true
		return result
	}

	for _, condition := range ks.Status.Conditions {
		if condition.Type == "Ready" {
			result.Status = condition.Message

			break
		}
	}

	if result.Status == "Release reconciliation succeeded" {
		result.IsError = false
		result.IsOkay = true
	} else if result.Status == "Install retries exhausted" {
		result.IsError = true
		result.IsOkay = false
	}

	return result
}
