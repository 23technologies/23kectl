package install

import (
	"context"
	"fmt"
	"path"

	"github.com/23technologies/23kectl/pkg/flux_utils"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func installVPACRDs(kubeconfigArgs *genericclioptions.ConfigFlags, kubeclientOptions *runclient.Options) error {
	if viper.GetBool("baseCluster.hasVerticalPodAutoscaler") {
		return nil
	}

	fmt.Println("Creating VPA CRDs")

	// todo move to kustomization (install from 23ke repo)
	dirPath := "./pkg/install/base-addons"
	filePath := path.Join(dirPath, "vpa-v1-crd-gen.yaml")

	result, err := utils.Apply(context.TODO(), kubeconfigArgs, kubeclientOptions, dirPath, filePath)

	fmt.Println(result)

	if err != nil {
		return err
	}

	return nil
}
