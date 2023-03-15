package install

import (
	"bytes"
	"context"
	"fmt"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func createBucketSecret(kubeClient client.Client) error {

	sec := corev1.Secret{
		TypeMeta: v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      common.BUCKET_SECRET_NAME,
			Namespace: common.FLUX_NAMESPACE,
		},
		Data: map[string][]byte{
			"accesskey": []byte(viper.GetString("bucket.accesskey")),
			"secretkey": []byte(viper.GetString("bucket.secretkey")),
		},
		Type: "Opaque",
	}

	err := Container.Create(context.Background(), &sec)
	if err != nil {
		err = kubeClient.Update(context.Background(), &sec)
		if err != nil {
			return err
		}
	}

	return nil

}

func create23keConfigSecret(kubeClient client.Client) error {
	fmt.Println("Creating '23ke-config' secret")

	tpl, err := makeTemplate().Parse(`
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
        email: {{ .Issuer.Acme.Email }}
    domains:
      global: # means used for ingress, gardener defaultDomain and internalDomain
        {{- nindent 8 (toYaml .DomainConfig) }}
`)

	if err != nil {
		return err
	}
	keConfig, err := getKeConfig()
	if err != nil {
		return err
	}

	buffer := bytes.Buffer{}
	err = tpl.Execute(&buffer, keConfig)
	if err != nil {
		return err
	}

	bytes := buffer.Bytes()
	_23keConfigSec := corev1.Secret{}

	err = k8syaml.Unmarshal(bytes, &_23keConfigSec)
	if err != nil {
		return err
	}

	err = Container.Create(context.Background(), &_23keConfigSec)
	if err != nil {
		err = kubeClient.Update(context.Background(), &_23keConfigSec)
		if err != nil {
			return err
		}
	}

	return nil
}
