package check

import (
	"context"
	"github.com/fluxcd/kustomize-controller/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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

	err := KubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}, ks)

	if err != nil {
		result.IsError = true
		return result
	}

	result.Status = getMessage(ks.Status.Conditions, "Ready")

	if strings.Contains(result.Status, "Applied revision") {
		result.IsError = false
		result.IsOkay = true
	} else if result.Status == "SOME DEFINITIVE ERROR" {
		result.IsError = true
		result.IsOkay = false
	}

	return result
}
