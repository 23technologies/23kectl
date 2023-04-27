package check

import (
	"context"
	"regexp"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// parse status from current helm releaes
	// we assume that we can build our logic on
	// this status message
	for _, condition := range hr.Status.Conditions {
		if condition.Type == "Ready" {
			result.Status = condition.Message
			break
		}
	}

	regexMap := map[*regexp.Regexp]func(res *Result, matches []string){}
	regexMap[regexp.MustCompile("Release reconciliation succeeded")] = func(res *Result, matches []string) { res.IsError = false; res.IsOkay = true }
	regexMap[regexp.MustCompile("install retries exhausted|Helm install failed")] = func(res *Result, matches []string) { res.IsError = true; res.IsOkay = false }
	// https://regex101.com/r/1l7ita/1
	regexMap[regexp.MustCompile("^HelmChart '(?P<namespace>.*)/(?P<name>.*)' is not ready$")] = handleHelmChartError

	for curRegexp, curFunc := range regexMap {
		matches := curRegexp.FindStringSubmatch(result.Status)
		if matches != nil {
			curFunc(result, matches)
			break
		}
	}
	return result
}

func handleHelmChartError(res *Result, matches []string) {
	namespace := matches[1]
	name := matches[2]

	hc := &sourcev1.HelmChart{}

	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, hc)

	if err != nil {
		res.Status = res.Status + ": " + err.Error()
	} else {
		hcReadyMessage := getMessage(hc.Status.Conditions, "Ready")
		newline := "\n  > "
		res.Status = res.Status + newline + strings.Replace(hcReadyMessage, ": ", newline, -1)
	}

	res.IsError = true
	res.IsOkay = false
}
