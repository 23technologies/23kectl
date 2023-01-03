package install

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/23technologies/23kectl/pkg/common"
	"github.com/spf13/viper"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"
)

func newDomainConfigAzure(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAzure
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    common.DNS_PROVIDER_AZURE_DNS,
		Credentials: &dnsCredentials,
	}
}

func newDomainOSDesignate(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsOSDesignate
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    common.DNS_PROVIDER_OPENSTACK_DESIGNATE,
		Credentials: &dnsCredentials,
	}
}

func newDomainAWS53(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAWS53
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    common.DNS_PROVIDER_AWS_ROUTE_53,
		Credentials: &dnsCredentials,
	}
}

func createDomainConfiguration(domain string, dnsProvider string) (domainConfiguration, error) {

	switch dnsProvider {
	case common.DNS_PROVIDER_AZURE_DNS:
		return newDomainConfigAzure(domain), nil
	case common.DNS_PROVIDER_OPENSTACK_DESIGNATE:
		return newDomainOSDesignate(domain), nil
	case common.DNS_PROVIDER_AWS_ROUTE_53:
		return newDomainAWS53(domain), nil
	}

	return domainConfiguration{}, fmt.Errorf("input invalid for domain configuration")
}

var funcMap template.FuncMap

func getFuncMap() template.FuncMap {
	if funcMap == nil {
		funcMap = sprig.FuncMap()
		funcMap["toYaml"] = func(in interface{}) string {
			result, err := yaml.Marshal(in)
			if err != nil {
				panic("Error during marshaling")
			}
			return string(result)
		}
	}

	return funcMap
}

func makeTemplate() *template.Template {
	tpl := new(template.Template).Funcs(getFuncMap())

	return tpl
}

var configTemplate *template.Template

func getConfigTemplate() (*template.Template, error) {
	if configTemplate == nil {
		templateRoot := "templates/config"

		tpl := makeTemplate()

		s3Client, err := minio.New(viper.GetString("bucket.endpoint"), &minio.Options{
			Creds:  credentials.NewStaticV4(viper.GetString("bucket.accesskey"), viper.GetString("bucket.secretkey"), ""),
			Secure: true,
		})
		if err != nil {
			return nil, err
		}

		opts := minio.ListObjectsOptions{
			Prefix:    templateRoot,
			Recursive: true,
		}
		objList := s3Client.ListObjects(context.Background(), viper.GetString("version"), opts)

		for item := range objList {

			obj, err := s3Client.GetObject(context.Background(), viper.GetString("version"),
				item.Key, minio.GetObjectOptions{})
			if err != nil {
				return nil, err
			}
			stat, err := obj.Stat()

			name := strings.Replace(item.Key, templateRoot+"/", "", 1)
			content := make([]byte, stat.Size)
			obj.Read(content)

			_, err = tpl.New(name).Parse(string(content))
			if err != nil {
				return nil, err
			}
			configTemplate = tpl
		}

	}
	return configTemplate, nil
}

func writeConfigDir(filesystem billy.Filesystem, gitRoot string) error {
	keConfig, err := getKeConfig()
	if err != nil {
		return err
	}

	configTemplate, err := getConfigTemplate()
	if err != nil {
		return err
	}

	for _, tpl := range configTemplate.Templates() {
		name := tpl.Name()

		destPath := path.Join(gitRoot, name)
		destDir := path.Dir(destPath)

		err := filesystem.MkdirAll(destDir, os.ModeDir|0700)
		if err != nil {
			return err
		}

		file, err := filesystem.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}

		err = tpl.Execute(file, keConfig)
		file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
