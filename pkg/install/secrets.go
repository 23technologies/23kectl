package install

import (
	"bytes"
	"context"
	"fmt"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

const templateString = `
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
`

func createBucketSecret(kubeClient client.WithWatch) error {

	if !viper.IsSet("bucket.accesskey") {
		prompt := &survey.Input{
			Message: "Please enter the accesskey, you got from 23T. This is part of your 23ke license.",
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("bucket.accesskey", queryResult)
	}

	if !viper.IsSet("bucket.secretkey") {
		prompt := &survey.Input{
			Message: "Please enter the secretkey, you got from 23T. This is part of your 23ke license.",
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("bucket.secretkey", queryResult)
	}

	sec := corev1.Secret{
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

	err := kubeClient.Create(context.Background(), &sec)
	if err != nil {
		err = kubeClient.Update(context.Background(), &sec)
		if err != nil {
			return err
		}
	}

	return nil

}

func create23keConfigSecret(kubeClient client.WithWatch) error {
	// Create the 23ke-config secret
	if !viper.IsSet("issuer.acme.email") {
		prompt := &survey.Input{
			Message: "Please enter your email address for acme certificate generation",
			Default: viper.GetString("admin.email"),
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required,email"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("issuer.acme.email", queryResult)
		err = viper.WriteConfig()
		if err != nil {
			return err
		}
	}

	if !viper.IsSet("domainConfig") {
		domainConfig, err := queryDomainConfig()
		if err != nil {
			return err
		}
		viper.Set("domainConfig", domainConfig)
		err = viper.WriteConfig()
		if err != nil {
			return err
		}
	}

	fmt.Println("Creating '23ke-config' secret")

	buffer := bytes.Buffer{}

	tpl, err := makeTemplate().Parse(templateString)
	if err != nil {
		return err
	}
	keConfig, err := getKeConfig()
	if err != nil {
		return err
	}

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

	err = kubeClient.Create(context.Background(), &_23keConfigSec)
	if err != nil {
		err = kubeClient.Update(context.Background(), &_23keConfigSec)
		if err != nil {
			return err
		}
	}

	return nil
}
