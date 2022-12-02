package install

import (
	"context"
	"fmt"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	corev1 "k8s.io/api/core/v1"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func generateDeployKey(kubeClient client.WithWatch, secretName string, repoUrl string) (*ssh.PublicKeys, error) {
	namespace := "flux-system"

	sec := corev1.Secret{}
	exists := false
	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}, &sec)
	if err == nil {
		exists = true
	}
	if exists {
		// todo check if exists AND works
		fmt.Println(`The following key was already deployed to you cluster and I did not change it. Make sure that your git repository can be accessed by this key.`)
		fmt.Println(string(sec.Data["identity.pub"]))

		key, _ := ssh.NewPublicKeys("git", sec.Data["identity"], "")
		return key, nil
	} else {
		fluxRepoSecret := corev1.Secret{}
		repourl, err := url.Parse(repoUrl)
		if err != nil {
			return nil, err
		}

		// define some options for the generation of the flux source secret
		sourceSecOpts := sourcesecret.MakeDefaultOptions()
		sourceSecOpts.PrivateKeyAlgorithm = "ed25519"
		sourceSecOpts.SSHHostname = repourl.Hostname()
		sourceSecOpts.Name = secretName

		// generate the flux source secret manifest and store it as []byte in the shootResources
		secManifest, err := sourcesecret.Generate(sourceSecOpts)
		_panic(err)
		// lastly, also deploy the flux source secret into the projectNamespace in the seed cluster
		// in order to reuse it, when other shoots are created
		err = k8syaml.Unmarshal([]byte(secManifest.Content), &fluxRepoSecret)

		_panic(err)
		fluxRepoSecret.SetNamespace(namespace)

		fmt.Println(`I created the following ssh key for you. Make sure that your git repository can be accessed by this key.`)
		fmt.Println(fluxRepoSecret.StringData["identity.pub"])
		kubeClient.Create(context.Background(), &fluxRepoSecret)

		key, _ := ssh.NewPublicKeys("git", fluxRepoSecret.Data["identity"], "")

		pressEnterToContinue()

		return key, nil
	}

}
