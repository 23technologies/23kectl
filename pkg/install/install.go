package install

import (
	"context"
	"text/template"
	"fmt"
	"bytes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// install ...

func Install(kubeconfig string)  {
  
	fmt.Println("Our install code here")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig )

	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

  test := &bytes.Buffer{}
	
	tpl := template.New("global")
	tpl.Parse(gardenerConfig.StringData["values.yaml"])
	tpl.Execute(test, map[string]string{"clusterIP": "10.1.0.1"})
	gardenerConfig.StringData["values.yaml"] = test.String()


	sec, err := clientset.CoreV1().Secrets("flux-system").Create(context.Background(), &gardenerConfig, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
	}
	_ = sec
	
}
