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

	// define a map from regexp to function
	// if we find a match we process the event by an appropriate function
	// this is assumed to stay branchless in the future which enables easy extensibility
	regexMap := map[*regexp.Regexp]func(res *Result, matches []string){}
	regexMap[regexp.MustCompile("Release reconciliation succeeded")] = func(res *Result, matches []string) {
		res.Status = prettify(matches[0])
		res.IsError = false
		res.IsOkay = true
	}
	regexMap[regexp.MustCompile("install retries exhausted|upgrade retries exhausted|Helm install failed")] = func(res *Result, matches []string) {
		res.Status = prettify(matches[0])
		res.IsError = true
		res.IsOkay = false
	}
	regexMap[regexp.MustCompile("^Helm upgrade failed.*")] = func(res *Result, matches []string) {
		res.Status = prettify(matches[0])
		res.IsError = true
		res.IsOkay = false
	}
	regexMap[regexp.MustCompile("^HelmChart '(?P<namespace>.*)/(?P<name>.*)' is not ready$")] = handleHelmChartError // https://regex101.com/r/1l7ita/1
	regexMap[regexp.MustCompile("Helm test failed: pod (?P<podName>.*) failed")] = handeHelmTestError

	// iterate over status conditions in the helm releases
	// here all useful information about potential errors should be found
	for _, condition := range hr.GetConditions() {
		for curRegexp, curFunc := range regexMap {
			matches := curRegexp.FindStringSubmatch(condition.Message)
			if matches != nil {
				curFunc(result, matches)
				//	return result
			}
		}
	}

	return result
}

// handeHelmTestError ...
func handeHelmTestError(res *Result, matches []string) {

	// It seems controller-runtime does not allow to access the logs.
	// Use kubectl directly for the moment.
	var log bytes.Buffer
	cmd := exec.Command("kubectl", "logs", "-n", "garden", matches[1])
	cmd.Stdout = &log
	err := cmd.Run()

	if err != nil {
		panic(err)
	}

	// Do some easy formatting for the moment.
	// We should definitely look for some package doing the job in the end.
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
	res.Status = matches[0] + newline + replacer.Replace(wordwrap.WrapString(strings.TrimSpace(log.String()), 100))

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

	status := matches[0]

	if err != nil {
		status = status + ": " + err.Error()
	} else {
		hcReadyMessage := getMessage(hc.Status.Conditions, "Ready")
		status = status + ": " + hcReadyMessage
	}

	res.Status = prettify(status)
	res.IsError = true
	res.IsOkay = false
}

func prettify(message string) string {
	newline := "\n  > "
	return strings.Replace(message, ": ", newline, -1)
}
