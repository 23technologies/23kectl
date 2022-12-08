package install

import (
	"context"
	"fmt"
	"github.com/23technologies/23kectl/pkg/common"
	"net/url"
	"strings"

	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func generateDeployKey(kubeClient client.WithWatch, secretName string, repoUrl string) (*ssh.PublicKeys, error) {
	namespace := "flux-system"

	sec := corev1.Secret{}
	err := kubeClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}, &sec)
	exists := err == nil

	var keys *ssh.PublicKeys

	if exists {
		keys, _ = ssh.NewPublicKeys("git", sec.Data["identity"], "")

		fmt.Println(`A key was already deployed to your cluster and I did not change it.`)

		blockUntilKeyCanRead(repoUrl, keys, string(sec.Data["identity.pub"]))

		return keys, nil
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
		common.Panic(err)
		// lastly, also deploy the flux source secret into the projectNamespace in the seed cluster
		// in order to reuse it, when other shoots are created
		err = k8syaml.Unmarshal([]byte(secManifest.Content), &fluxRepoSecret)

		common.Panic(err)
		fluxRepoSecret.SetNamespace(namespace)

		fmt.Println(`I created an ssh key for you.`)

		kubeClient.Create(context.Background(), &fluxRepoSecret)

		keys, _ = ssh.NewPublicKeys("git", fluxRepoSecret.Data["identity"], "")

		blockUntilKeyCanRead(repoUrl, keys, string(fluxRepoSecret.Data["identity.pub"]))

		return keys, nil
	}
}

func blockUntilKeyCanRead(repoUrl string, keys *ssh.PublicKeys, pubkey string) {
	var err error
	for {
		err = keyCanRead(repoUrl, keys)
		if err == nil {
			fmt.Println(`Read access granted.`)
			break
		}

		err := fmt.Errorf("make sure that %s can be accessed by this key:\n%s", repoUrl, err)
		fmt.Println(err.Error())

		common.PrintHighlight(strings.TrimSpace(pubkey))
		common.PressEnterToContinue()
	}
}

func keyCanRead(url string, publicKeys *ssh.PublicKeys) error {
	remote := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	})

	_, err := remote.List(&git.ListOptions{
		Auth: publicKeys,
	})

	if err != nil && err != transport.ErrEmptyRemoteRepository {
		return err
	}
	return nil
}
