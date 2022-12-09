package install

import (
	"context"
	"errors"
	"fmt"
	"github.com/23technologies/23kectl/pkg/logger"
	"time"

	"github.com/23technologies/23kectl/pkg/common"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fluxcd/pkg/apis/meta"
	sourcecontrollerv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createGitRepositories(kubeClient client.WithWatch, keys *ssh.PublicKeys) error {
	log := logger.Get("createGitRepositories")
	var err error

	if !viper.IsSet("version") {
		tags, err := common.List23keTags(keys)
		if err != nil {
			return err
		}
		prompt := &survey.Select{
			Message: "Select the 23ke version you want to install",
			Options: tags,
		}
		var queryResult string
		err = survey.AskOne(prompt, &queryResult, withValidator("required"))
		exitOnCtrlC(err)
		if err != nil {
			return err
		}
		viper.Set("version", queryResult)
		viper.WriteConfig()
	}

	tag := viper.GetString("version")

	gitrepo23ke := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.BASE_23KE_GITREPO_NAME,
			Namespace: common.FLUX_NAMESPACE,
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			URL:       common.BASE_23KE_GITREPO_URI,
			SecretRef: &meta.LocalObjectReference{Name: common.BASE_23KE_GITREPO_KEY},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Tag: tag},
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	err = kubeClient.Create(context.TODO(), &gitrepo23ke, &client.CreateOptions{})
	if err != nil {
		log.Info("Couldn't create git source "+common.BASE_23KE_GITREPO_NAME, "error", err)
	}

	gitRepoUrl := viper.GetString("admin.gitrepourl")
	gitRepoBranch := viper.GetString("admin.gitrepobranch")

	gitrepo23keconfig := sourcecontrollerv1beta2.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "source.toolkit.fluxcd.io/v1beta2",
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.CONFIG_23KE_GITREPO_NAME,
			Namespace: common.FLUX_NAMESPACE,
		},
		Spec: sourcecontrollerv1beta2.GitRepositorySpec{
			URL:       gitRepoUrl,
			SecretRef: &meta.LocalObjectReference{Name: common.CONFIG_23KE_GITREPO_KEY},
			Interval:  metav1.Duration{Duration: time.Minute},
			Reference: &sourcecontrollerv1beta2.GitRepositoryRef{Branch: gitRepoBranch},
		},
		Status: sourcecontrollerv1beta2.GitRepositoryStatus{},
	}

	err = kubeClient.Create(context.TODO(), &gitrepo23keconfig, &client.CreateOptions{})
	if err != nil {
		log.Info("Couldn't create git source "+common.CONFIG_23KE_GITREPO_NAME, "error", err)
	}
	return nil
}

func updateConfigRepo(publicKeys ssh.PublicKeys) error {
	log := logger.Get("updateConfigRepo")
	gitRepo := viper.GetString("admin.gitrepourl")

	var err error
	workTreeFs := memfs.New()

	fmt.Printf("Cloning config repo to memory\n")
	repository, err := git.Clone(memory.NewStorage(), workTreeFs, &git.CloneOptions{
		URL:        gitRepo,
		Auth:       &publicKeys,
		NoCheckout: true,
	})
	if err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		panic(err)
	}

	branchName := viper.GetString("admin.gitRepoBranch")

	// check whether the remote reference exists
	// if not, we create an orphaned branch, this is the same as git init would do and should
	// https://github.com/go-git/go-git/issues/370
	remoteRef, err := repository.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
	if err != nil {
		repository.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branchName)))
	}

	worktree, err := repository.Worktree()
	if err != nil {
		log.Info("Couldn't get worktree", "error", err)
	}

	// if the remoteRef was found we either checkout and create a local copy of the remote branch,
	// or, if the local branch already exists, we simply check it out
	if remoteRef != nil {
		_, err = repository.Reference(plumbing.NewBranchReferenceName(branchName), true)
		if err != nil {
			err = worktree.Checkout(&git.CheckoutOptions{
				Hash:   remoteRef.Hash(),
				Branch: plumbing.NewBranchReferenceName(branchName),
				Create: true,
			})
			if err != nil {
				return err
			}
		} else {
			err = worktree.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(branchName),
			})
			if err != nil {
				return err
			}
		}
	}

	_, _ = worktree.Remove(".")

	fmt.Printf("Writing new config\n")

	err = writeConfigDir(workTreeFs, ".")
	if err != nil {
		return err
	}

	_, _ = worktree.Add(".")
	status, _ := worktree.Status()

	if status.IsClean() {
		log.Info("Worktree is clean. Not committing anything.")
	} else {
		log.Info("Commiting to config repo")
		_, err = worktree.Commit("Config update through 23kectl", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "23ke Ctl",
				Email: "23kectl@23technologies.cloud",
				When:  time.Now(),
			},
		})
		if err != nil {
			return err
		}

		log.Info("Pushing to config repo")
		err = repository.Push(&git.PushOptions{
			Auth: &publicKeys,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
