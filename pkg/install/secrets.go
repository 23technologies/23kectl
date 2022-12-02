package install

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func create23keConfigSecret(keConfig *KeConfig, kubeClient client.WithWatch) {
	// Create the 23ke-config secret
	fmt.Println("Creating '23ke-config' secret")
	filePath := path.Join("/tmp", "23ke-config.yaml")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		file.Close()
		panic(err)
	}
	err = getLocalTemplate().ExecuteTemplate(file, "23ke-config.yaml", keConfig)
	file.Close()
	_panic(err)

	_23keConfigSec := corev1.Secret{}
	tmpByte, err := os.ReadFile(file.Name())
	k8syaml.Unmarshal(tmpByte, &_23keConfigSec)
	kubeClient.Create(context.Background(), &_23keConfigSec)
}
