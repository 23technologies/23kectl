package install

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var gardenerConfig = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name: "gardener-values",
		Namespace: "flux-system",
	},
	StringData: map[string]string{
		"values.yaml":
`global:
  deployment:
   virtualGarden:
     clusterIP: {{ .clusterIP }}`,
	},
	Type: "Opaque",
}
