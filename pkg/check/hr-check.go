package check

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/mitchellh/go-wordwrap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	err := KubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}, hr)

	if err != nil {
		result.IsError = true
		return result
	}

	// define a slice of handler including a regexp and a function
	// if we find a match we process the event by an appropriate function
	// this is assumed to stay branchless in the future which enables easy extensibility
	// the order of processing is important as we prioritize the status messages
	type handler struct {
		regex *regexp.Regexp
		fn    func(name string, res *Result, matches []string)
	}

	handlers := []handler{
		{
			regex: regexp.MustCompile("Helm install failed: timed out waiting for the condition"),
			fn:    handleHelmInstallTimeoutError,
		},
		{
			regex: regexp.MustCompile("Helm test failed: pod (?P<podName>.*) failed"),
			fn:    handeHelmTestError,
		},
		{
			regex: regexp.MustCompile("(install retries exhausted|upgrade retries exhausted|Helm install failed|Helm upgrade failed).*"),
			fn: func(name string, res *Result, matches []string) {
				res.Status = prettify(matches[0])
				res.IsError = true
				res.IsOkay = false
			},
		},
		{
			regex: regexp.MustCompile("^HelmChart '(?P<namespace>.*)/(?P<name>.*)' is not ready$"),
			fn:    handleHelmChartError,
		},
		{
			regex: regexp.MustCompile("Release reconciliation succeeded"),
			fn: func(name string, res *Result, matches []string) {
				res.Status = prettify(matches[0])
				res.IsError = false
				res.IsOkay = true
			},
		},
	}

	// iterate over status conditions in the helm releases
	// here all useful information about potential errors should be found
	for _, curHandler := range handlers {
		for _, condition := range hr.GetConditions() {
			matches := curHandler.regex.FindStringSubmatch(condition.Message)
			if matches != nil {
				curHandler.fn(hr.Name, result, matches)
				return result
			}
		}
	}

	return result
}

// handeHelmTestError ...
func handeHelmTestError(name string, res *Result, matches []string) {

	// It seems controller-runtime does not allow to access the logs.
	// Use kubectl directly for the moment.
	test := KubeClientGo.CoreV1().Pods("garden").GetLogs(matches[1], &corev1.PodLogOptions{})
	logs, err := test.Do(context.Background()).Raw()
	log := string(logs)
	if err != nil {
		log = fmt.Sprintf("couldn't get pod logs: %s", err)
	}

	res.Status = matches[0] + indent(wordwrap.WrapString(strings.TrimSpace(log), 100), 4)
	res.IsError = true
	res.IsOkay = false
}

func handleHelmInstallTimeoutError(name string, res *Result, matches []string) {

	// implement further checks here by adding other cases
	// todo: define a cleaner interface for this process
	switch name {
	case "internal-gardenlet":
		test, _ := KubeClientGo.CoreV1().Pods("garden").List(context.Background(), metav1.ListOptions{
			LabelSelector: "role=gardenlet,app=gardener",
		})
		var log string
		for _, pod := range test.Items {
			logs, _ := KubeClientGo.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Do(context.Background()).Raw()
			if !strings.Contains(log, string(logs)) {
				log += "\n" + string(logs)
			}
		}

		res.Status = matches[0] + indent(wordwrap.WrapString(strings.TrimSpace(log), 100), 4)
		res.IsError = true
		res.IsOkay = false
	default:
		res.Status = prettify(matches[0])
		res.IsError = true
		res.IsOkay = false
	}
}

func handleHelmChartError(name string, res *Result, matches []string) {
	namespace := matches[1]
	podName := matches[2]

	hc := &sourcev1.HelmChart{}

	err := KubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      podName,
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

// indent ...
func indent(in string, n int) string {

	// Do some easy formatting for the moment.
	// We should definitely look for some package doing the job in the end.
	in = "\n" + in
	var replacement = "\n" + strings.Repeat(" ", n) + "> "
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

	return replacer.Replace(in)

}
