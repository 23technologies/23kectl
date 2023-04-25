package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/fluxcd/flux2/pkg/manifestgen"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/23technologies/23kectl/pkg/logger"
	"github.com/fatih/color"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utils "github.com/23technologies/23kectl/pkg/fluxutils"
	"github.com/itchyny/json2yaml"
)

var Container = struct {
	BlockUntilKeyCanRead func(string, *ssh.PublicKeys, string)
	GetSSHHostname       func(_ *url.URL) string
	QueryConfigKey       func(configKey string, _ func() (any, error)) error
	CreateFluxManifest   func() (*manifestgen.Manifest, error)
	Apply                func(ctx context.Context, rcg genericclioptions.RESTClientGetter, opts *runclient.Options, root, manifestPath string) (string, error)
	Create               func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
}{
	BlockUntilKeyCanRead: blockUntilKeyCanRead,
	GetSSHHostname:       getSSHHostname,
	QueryConfigKey:       common.QueryConfigKey,
	CreateFluxManifest:   createFluxManifest,
	Apply:                utils.Apply,
}

func Install(kubeconfig string, isDryRun bool) error {
	watch()

	return nil

	log := logger.Get("Install")

	keConfiguration := &KeConfig{}
	UnmarshalKeConfig(keConfiguration)

	var err error
	kubeconfigArgs, kubeclientOptions, kubeClient, err := common.CreateKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	Container.Create = kubeClient.Create

	err = queryConfig(kubeClient)
	if err != nil {
		return err
	}
	UnmarshalKeConfig(keConfiguration)

	// initialize container
	// This is espcially important when running in dry run mode
	if isDryRun {
		Container.Apply = applyDryRun
		Container.Create = create
		Container.GetSSHHostname = func(_ *url.URL) string { return "github.com" }
		Container.BlockUntilKeyCanRead = func(_ string, _ *ssh.PublicKeys, _ string) {}

		gitRepoUrl := viper.GetString("admin.gitrepourl")
		if !strings.Contains(gitRepoUrl, "file://") {
			return fmt.Errorf("dry run mode only supports local git repositories. I have written a config file for you. If you just wanted to craft an inital config file, you can ignore this error")
		}
		gitRepoPath := strings.SplitAfter(gitRepoUrl, "//")[1]
		_, err = git.PlainInit(gitRepoPath, true)
		if err != nil {
			return err
		}
	}

	fmt.Println("Installing flux")
	err = installFlux(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		log.Error(err, "Couldn't install flux")
		return err
	}

	err = createBucketSecret(kubeClient)
	if err != nil {
		return err
	}

	fmt.Println("Generating 23ke-config deploy key")
	fmt.Println(`You will need to add this key to your git remote git repository.`)
	common.PrintWarn("This key needs write access!")
	publicKeysConfig, err := generateDeployKey(kubeClient, common.CONFIG_23KE_GITREPO_KEY, viper.GetString("admin.gitrepourl"))
	if err != nil {
		return err
	}

	err = create23keConfigSecret(kubeClient)
	if err != nil {
		return err
	}

	err = create23keBucket(kubeClient)
	if err != nil {
		return err
	}

	err = createGitRepositories(kubeClient)
	if err != nil {
		return err
	}

	err = createKustomizations(kubeClient)
	if err != nil {
		return err
	}

	err = updateConfigRepo(publicKeysConfig)
	if err != nil {
		return err
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("Awesome. Your gardener installation should be up within 10 minutes.")
	fmt.Printf("Once it's done you can login as %s.\n", color.BlueString(keConfiguration.Admin.Email))
	fmt.Printf("Go kill some time by eagerly pressing F5 on https://dashboard.%s\n", color.BlueString(keConfiguration.DomainConfig.Domain))

	return nil
}

func getKeConfig() (*KeConfig, error) {
	keConfig := new(KeConfig)
	err := UnmarshalKeConfig(keConfig)
	if err != nil {
		return nil, nil
	}

	return keConfig, nil
}

// unmarshalKeConfig ...
func UnmarshalKeConfig(config *KeConfig) error {
	err := viper.Unmarshal(config)
	if err != nil {
		return err
	}

	_, ok := (config.DomainConfig.Credentials).(map[string]interface{})
	if ok {
		var creds interface{}
		switch config.DomainConfig.Provider {
		case common.DNS_PROVIDER_AZURE_DNS:
			creds = dnsCredentialsAzure{}
		case common.DNS_PROVIDER_OPENSTACK_DESIGNATE:
			creds = dnsCredentialsOSDesignate{}
		case common.DNS_PROVIDER_AWS_ROUTE_53:
			creds = dnsCredentialsAWS53{}
		}
		err = mapstructure.Decode(config.DomainConfig.Credentials, &creds)
		if err != nil {
			return err
		}
		config.DomainConfig.Credentials = creds
	}

	_, ok = (config.BackupConfig.Credentials).(map[string]interface{})
	if ok {
		var creds interface{}
		switch config.BackupConfig.Provider {
		case common.BUCKET_PROVIDER_AZURE:
			creds = backupCredentialsAzure{}
		}
		err = mapstructure.Decode(config.BackupConfig.Credentials, &creds)
		if err != nil {
			return err
		}

		config.BackupConfig.Credentials = creds
	}
	return nil
}

// create ...
// implements a dry run create function. It only outputs the objects
// which would be created in the cluster
func create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	tmp, err := json.Marshal(obj)
	jsonReader := bytes.NewReader(tmp)
	yamlWriter := strings.Builder{}
	json2yaml.Convert(&yamlWriter, jsonReader)
	if err != nil {
		panic(err)
	}
	fmt.Println("---")
	fmt.Println(yamlWriter.String())
	return nil
}
