package install

import (
	"bytes"
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func create23keConfigSecret(kubeClient client.WithWatch) {
	// Create the 23ke-config secret
	if !viper.IsSet("issuer.acme.email") {
		prompt := &survey.Input{
			Message: "Please enter your email address for acme certificate generation",
			Default: viper.GetString("admin.email"),
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required,email"))
		viper.Set("issuer.acme.email", queryResult)
		viper.WriteConfig()
		handleErr(err)
	}

	if !viper.IsSet("domainConfig") {
		viper.Set("domainConfig", queryDomainConfig())
		viper.WriteConfig()
	}
	
	fmt.Println("Creating '23ke-config' secret")

	buffer := bytes.Buffer{}
	err  := getLocalTemplate().ExecuteTemplate(&buffer, "23ke-config.yaml", getKeConfig())
	_panic(err)

	bytes := buffer.Bytes()
	_23keConfigSec := corev1.Secret{}
	k8syaml.Unmarshal(bytes, &_23keConfigSec)
	err = kubeClient.Create(context.Background(), &_23keConfigSec)
	_panic(err)
}
