package install

import (
	"context"
	"errors"
	"fmt"
	"github.com/fluxcd/pkg/apis/meta"
	sourcecontrollerv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func createGitRepositories(kubeClient client.WithWatch, keConfiguration KeConfig) {
	var err error

	gitrepo23ke := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke",
			Namespace: "flux-system",
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			URL:       _23KERepoURI,
			SecretRef: &meta.LocalObjectReference{Name: "23ke-key"},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Tag: keConfiguration.Version},
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	err = kubeClient.Create(context.TODO(), &gitrepo23ke, &client.CreateOptions{})
	if err != nil {
		printErr(err)
	}

	gitrepo23keconfig := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "23ke-config",
			Namespace: "flux-system",
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			URL:       keConfiguration.GitRepo,
			SecretRef: &meta.LocalObjectReference{Name: "23ke-config-key"},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Branch: "main"}, // todo ask user for branch
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	err = kubeClient.Create(context.TODO(), &gitrepo23keconfig, &client.CreateOptions{})
	if err != nil {
		printErr(err)
	}
}

func updateConfigRepo(keConfig *KeConfig, publicKeys ssh.PublicKeys) error {
	var err error
	workTreeFs := memfs.New()

	// todo catch "empty repo" error
	fmt.Printf("Cloning config repo to memory\n")
	repository, err := git.Clone(memory.NewStorage(), workTreeFs, &git.CloneOptions{
		Auth: &publicKeys,
		URL:  keConfig.GitRepo,
	})
	if err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		panic(err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		printErr(err)
	}

	_, err = worktree.Remove(".")
	if err != nil {
		printErr(err)
	}

	fmt.Printf("Writing new config\n")
	err = writeConfigDir(workTreeFs, ".", keConfig)
	if err != nil {
		printErr(err)
	}

	_, err = worktree.Add(".")
	if err != nil {
		printErr(err)
	}

	status, err := worktree.Status()
	if err != nil {
		printErr(err)
	}

	if status.IsClean() {
		fmt.Printf("Git reports no changes to config repo\n")
	} else {
		fmt.Printf("Commiting to config repo\n")
		_, err = worktree.Commit("Config update through 23kectl", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "23ke Ctl",
				Email: "23kectl@23technologies.cloud",
				When:  time.Now(),
			},
		})
		if err != nil {
			printErr(err)
		}

		fmt.Printf("Pushing to config repo\n")
		err = repository.Push(&git.PushOptions{
			Auth: &publicKeys,
		})
		printErr(err)
	}

	return nil
}
