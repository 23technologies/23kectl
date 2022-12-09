package install

import (
	"context"
	"fmt"
	"github.com/23technologies/23kectl/pkg/flux_utils"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func installFlux(kubeClient client.WithWatch, kubeconfigArgs *genericclioptions.ConfigFlags, kubeclientOptions *runclient.Options) error {
	// Install flux.
	// We just copied over github.com/fluxcd/flux2/internal/utils to 23kectl/pkg/utils
	// and use the Apply function as is
	fmt.Println("Installing flux")

	tmpDir, err := manifestgen.MkdirTempAbs("", *kubeconfigArgs.Namespace)
	if err != nil {
		return err
	}

	defer os.RemoveAll(tmpDir)

	opts := install.MakeDefaultOptions()
	manifest, err := install.Generate(opts, "")
	if err != nil {
		return err
	}

	_, err = manifest.WriteFile(tmpDir)
	if err != nil {
		return err
	}

	_, err = utils.Apply(context.Background(), kubeconfigArgs, kubeclientOptions, tmpDir, path.Join(tmpDir, manifest.Path))
	if err != nil {
		return err
	}

	return nil
}
