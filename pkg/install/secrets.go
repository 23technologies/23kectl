package install

import (
	"context"
	"fmt"
	"os"
	"path"

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
	filePath := path.Join("/tmp", "23ke-config.yaml")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		file.Close()
		panic(err)
	}

	err = getLocalTemplate().ExecuteTemplate(file, "23ke-config.yaml", getKeConfig())
	file.Close()
	_panic(err)

	_23keConfigSec := corev1.Secret{}
	tmpByte, err := os.ReadFile(file.Name())
	k8syaml.Unmarshal(tmpByte, &_23keConfigSec)
	kubeClient.Create(context.Background(), &_23keConfigSec)
}
