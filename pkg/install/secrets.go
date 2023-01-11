package install

import (
	"bytes"
	"context"
	"fmt"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/AlecAivazis/survey/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

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

	if !viper.IsSet("version") {
		prompt := &survey.Input{
			Message: "Please enter the version to install.",
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("version", queryResult)
	}

	if !viper.IsSet("bucket.endpoint") {
		prompt := &survey.Input{
			Message: "Please enter the bucket endpoint, you got from 23T. This is part of your 23ke license.",
		}
		var queryResult string
		err := survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("bucket.endpoint", queryResult)
	}

	fmt.Println("Creating '23ke-config' secret")

	s3Client, err := minio.New(viper.GetString("bucket.endpoint"), &minio.Options{
		Creds:  credentials.NewStaticV4(viper.GetString("bucket.accesskey"), viper.GetString("bucket.secretkey"), ""),
		Secure: true,
	})
	if err != nil {
		return err
	}

	obj, err := s3Client.GetObject(context.Background(), viper.GetString("version"),
		"templates/23ke-config.yaml", minio.GetObjectOptions{})
	if err != nil {
		return err
	}

	stat, err := obj.Stat()
	if err != nil {
		return err
	}

	content := make([]byte, stat.Size)
	obj.Read(content)

	tpl, err := makeTemplate().Parse(string(content))

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

	err = kubeClient.Create(context.Background(), &_23keConfigSec)
	if err != nil {
		err = kubeClient.Update(context.Background(), &_23keConfigSec)
		if err != nil {
			return err
		}
	}

	return nil
}
