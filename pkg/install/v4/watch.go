package install

import (
	"fmt"
	"github.com/23technologies/23kectl/pkg/check"
	"time"
)

func watch() {
	checks := []check.Check{
		&check.HelmReleaseCheck{Name: "addons", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "admission-provider-azure-application", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "admission-provider-azure-runtime", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "cert-management", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "cert-manager", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "cloudprofiles", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "dashboard-application", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "dashboard-runtime", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "dnsprovider", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "etcd", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "etcd-events", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "extensions", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "external-dns-management", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "garden-content", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "gardener-application", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "gardener-configuration", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "gardener-runtime", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "identity", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "ingress-nginx", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "internal-gardenlet", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "issuer", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "kube-apiserver", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "networking-calico", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "os-gardenlinux", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "os-ubuntu", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "pre-gardener-configuration", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "provider-azure", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "provider-hcloud", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "terminal-controller-manager-application", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "terminal-controller-manager-runtime", Namespace: "flux-system"},
		&check.HelmReleaseCheck{Name: "velero", Namespace: "flux-system"},

		&check.KustomizationCheck{Name: "23ke-base", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "23ke-config", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "23ke-env-config", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "23ke-env-garden-content", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "flux-system", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "gardener", Namespace: "flux-system"},
		&check.KustomizationCheck{Name: "pre-gardener", Namespace: "flux-system"},
	}

	for {
		fmt.Print("\033[H\033[2J")

		for _, c := range checks {
			result := c.Run()

			emoji := "⌛"

			if result.IsError {
				emoji = "❌"
			} else if result.IsOkay {
				emoji = "✔️"
			}

			fmt.Printf("%s %s status: %s\n", emoji, c.GetName(), result.Status)
		}

		time.Sleep(time.Second * 5)
	}
}
