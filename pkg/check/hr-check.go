package check

import (
	"context"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type HelmReleaseCheck struct {
	Name      string
	Namespace string
}

func (d *HelmReleaseCheck) GetName() string {
	return d.Name
}

func (d *HelmReleaseCheck) Run() *Result {
	result := &Result{}

	hr := &helmv2.HelmRelease{}

	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}, hr)

	if err != nil {
		result.IsError = true
		return result
	}

	for _, condition := range hr.Status.Conditions {
		if condition.Type == "Ready" {
			result.Status = condition.Message

			break
		}
	}

	if result.Status == "Release reconciliation succeeded" {
		result.IsError = false
		result.IsOkay = true
	} else if result.Status == "install retries exhausted" || strings.Contains(result.Status, "Helm install failed") {
		result.IsError = true
		result.IsOkay = false
	}

	return result
}
