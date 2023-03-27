package install_test

import (
	installv2 "github.com/23technologies/23kectl/pkg/install/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const FIXTURE = `alicloud:
    enabled: false
aws:
    enabled: false
azure:
    enabled: false
citycloud:
    enabled: false
fugacloud:
    enabled: false
gcp:
    enabled: false
global:
    kubernetes:
        upstreamVersions:
            include: true
            versions:
                1.22.17:
                    classification: supported
                1.23.17:
                    classification: supported
                1.24.12:
                    classification: supported
                1.25.8:
                    classification: supported
        versions: {}
    seedSelector:
        enabled: false
        selector: {}
hcloud:
    enabled: false
pluscloud-open:
    enabled: false
regiocloud:
    enabled: false
scs-community-platform:
    enabled: false
wavestack:
    enabled: false
`

func init() {
	It("Parses cloudprofiles correctly", func() {
		expected := []string{
			"alicloud",
			"aws",
			"azure",
			"citycloud",
			"fugacloud",
			"gcp",
			"hcloud",
			"pluscloud-open",
			"regiocloud",
			"scs-community-platform",
			"wavestack",
		}

		result, err := installv2.ParseAvailableCloudProfiles([]byte(FIXTURE))

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(expected))
	})
}
