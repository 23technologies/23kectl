package install

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

func newDomainConfigAzure(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAzure
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    "azure-dns",
		Credentials: &dnsCredentials,
	}
}

func (d domainConfiguration) marshal() string {

	result, err := yaml.Marshal(d)
	if err != nil {
		panic("Error during marshaling")
	}

	return string(result)
}

func createDomainConfiguration(domain string, dnsProvider string) (domainConfiguration, error)  {

	switch dnsProvider {
	case "azure-dns":
		return newDomainConfigAzure(domain), nil
	}

	return domainConfiguration{}, fmt.Errorf("input invalid for domain configuration")
}

func keConfigTpl() []string {
	return []string {`
apiVersion: v1
kind: Secret
metadata:
  name: 23ke-config
  namespace: flux-system
type: Opaque
stringData:
  values.yaml: |
    clusterIdentity: {{ .ClusterIdentity }}
    dashboard:
      clientSecret: {{ .Dashboard.ClientSecret }} 
      sessionSecret: {{ .Dashboard.SessionSecret }}
    kubeApiServer:
      basicAuthPassword: {{ .KubeApiServer.BasicAuthPassword }}
    issuer:
      acme:
        email: {{ .EmailAddress }}
    domains:
      global: # means used for ingress, gardener defaultDomain and internalDomain
        {{- nindent 8 (toYaml .DomainConfig) }}
`,
`
apiVersion: v1
kind: Secret
metadata:
 name: gardener-values
 namespace: flux-system
type: Opaque
stringData:
 values.yaml: |
   global:
     deployment:
       virtualGarden:
         clusterIP: {{ .Gardener.ClusterIP }} 
`,
`
apiVersion: v1
kind: Secret
metadata:
 name: gardenlet-values
 namespace: flux-system
type: Opaque
stringData:
 values.yaml: |-
   config:
     seedConfig:
       metadata:
         name: initial-seed 
       spec:
         networks:
           nodes: {{ .Gardenlet.SeedNodeCidr }} 
           pods: {{ .Gardenlet.SeedPodCidr }} 
           services: {{ .Gardenlet.SeedServiceCidr }} 
           shootDefaults:
             pods: 100.73.0.0/16
             services: 100.88.0.0/13
         provider:
           region: {{ .BaseCluster.Region }} 
           type: {{ .BaseCluster.Provider }} 
         settings:
           verticalPodAutoscaler:
             enabled: false # enable if your base cluster does not 
`, 
`
apiVersion: v1
kind: Secret
metadata:
  name: extensions-values
  namespace: flux-system
type: Opaque
stringData:
  values.yaml: |
    os-ubuntu:
      enabled: true
    os-gardenlinux:
      enabled: true
    networking-calico:
      enabled: true
    {{- nindent 4 (toYaml .ExtensionsConfig) }}
`,
`apiVersion: v1
kind: Secret
metadata:
 name: dashboard-values
 namespace: flux-system
type: Opaque
stringData:
 values.yaml: |
   frontendConfig:
     seedCandidateDeterminationStrategy: MinimalDistance
`, 
`apiVersion: v1
  kind: Secret
  metadata:
    name: identity-values
    namespace: flux-system
  type: Opaque
  stringData:
    values.yaml: |
      staticPasswords:
      - email: {{ .emailAddress }} 
        hash: {{ bcrypt .AdminPassword }}  
        username: "admin"
        userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
`,
`apiVersion: v1
kind: Secret
metadata:
 name: cloudprofiles-values
 namespace: flux-system
type: Opaque
stringData:
 values.yaml: |
   global:
     kubernetes:
       versions:
         1.22.9:
           classification: preview
     seedSelector:
       enabled: true
       selector:
         providerTypes:
           - hcloud
   hcloud:
     enabled: true
   betacloud:
     enabled: true
`,
}
}
