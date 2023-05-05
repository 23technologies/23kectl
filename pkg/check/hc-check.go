package check

import (
	"context"
	v1 "github.com/fluxcd/source-controller/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HelmChartsCheck struct {
	Name      string
	Namespace string
}

func (d *HelmChartsCheck) GetName() string {
	return d.Name
}

func (d *HelmChartsCheck) Run() *Result {
	result := &Result{}

	hc := &v1.HelmChart{}

	err := KubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}, hc)

	if err != nil {
		result.IsError = true
		return result
	}

	result.Status = getMessage(hc.Status.Conditions, "Ready")

	if result.Status == "Applied revision" {
		result.IsError = false
		result.IsOkay = true
	} else if result.Status == "SOME DEFINITIVE ERROR" {
		result.IsError = true
		result.IsOkay = false
	}

	return result
}
