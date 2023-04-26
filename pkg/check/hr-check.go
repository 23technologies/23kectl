package check

import (
	"context"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"regexp"
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

	// https://regex101.com/r/1l7ita/1
	hcRegex := regexp.MustCompile("^HelmChart '(?P<namespace>.*)/(?P<name>.*)' is not ready$")
	matches := hcRegex.FindStringSubmatch(result.Status)

	if matches != nil {
		namespace := matches[hcRegex.SubexpIndex("namespace")]
		name := matches[hcRegex.SubexpIndex("name")]

		hc := &sourcev1.HelmChart{}

		err := kubeClient.Get(context.Background(), client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, hc)

		if err != nil {
			result.Status = result.Status + ": " + err.Error()
		} else {
			hcReadyMessage := getMessage(hc.Status.Conditions, "Ready")

			newline := "\n  > "

			result.Status = result.Status + newline + strings.Replace(hcReadyMessage, ": ", newline, -1)
		}

		result.IsError = true
		result.IsOkay = false
	}

	return result
}
