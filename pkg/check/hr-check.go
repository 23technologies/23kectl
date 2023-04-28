package check

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/mitchellh/go-wordwrap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	result.Conditions = *hr.GetStatusConditions()
	for _, condition := range hr.Status.Conditions {

		regexMap := map[*regexp.Regexp]func(res *Result, matches []string){}
		regexMap[regexp.MustCompile("Release reconciliation succeeded")] = func(res *Result, matches []string) { res.IsError = false; res.IsOkay = true }
		regexMap[regexp.MustCompile("install retries exhausted|upgrade retries exhausted|Helm install failed")] = func(res *Result, matches []string) { res.IsError = true; res.IsOkay = false }
		regexMap[regexp.MustCompile("^HelmChart '(?P<namespace>.*)/(?P<name>.*)' is not ready$")] = handleHelmChartError // https://regex101.com/r/1l7ita/1
		regexMap[regexp.MustCompile("Helm test failed: pod (?P<podName>.*) failed")] = handeHelmTestError

		for curRegexp, curFunc := range regexMap {
			matches := curRegexp.FindStringSubmatch(condition.Message)
			if matches != nil {
				curFunc(result, matches)
			}
		}

	}
	return result
}


// handeHelmTestError ...
func handeHelmTestError(res *Result, matches []string) {

	var log bytes.Buffer
	cmd := exec.Command("kubectl", "logs", "-n", "garden", matches[1])
	cmd.Stdout = &log
	cmd.Run()

	const replacement = "\n    > "

	var replacer = strings.NewReplacer(
    "\r\n", replacement,
    "\r", replacement,
    "\n", replacement,
    "\v", replacement,
    "\f", replacement,
    "\u0085", replacement,
    "\u2028", replacement,
    "\u2029", replacement,
	)

	newline := "\n  > "
	res.Status = res.Status + newline + matches[0] + newline + replacer.Replace(wordwrap.WrapString(log.String(), 100))

	res.IsError = true
	res.IsOkay = false
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
