package install

import (
	"context"
	"fmt"

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

	sec, err := clientset.CoreV1().Secrets("flux-system").Create(context.Background(), &gardenerConfig, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
	}
	_ = sec
	
}
