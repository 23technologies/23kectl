package install

import (
	"embed"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

func newDomainConfigAzure(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAzure
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    DNS_PROVIDER_AZURE_DNS,
		Credentials: &dnsCredentials,
	}
}

func newDomainOSDesignate(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsOSDesignate
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    DNS_PROVIDER_OPENSTACK_DESIGNATE,
		Credentials: &dnsCredentials,
	}
}

func newDomainAWS53(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAWS53
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    DNS_PROVIDER_AWS_ROUTE_53,
		Credentials: &dnsCredentials,
	}
}

func createDomainConfiguration(domain string, dnsProvider string) (domainConfiguration, error) {

	switch dnsProvider {
	case DNS_PROVIDER_AZURE_DNS:
		return newDomainConfigAzure(domain), nil
	case DNS_PROVIDER_OPENSTACK_DESIGNATE:
		return newDomainOSDesignate(domain), nil
	case DNS_PROVIDER_AWS_ROUTE_53:
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

//go:embed templates
var embedFS embed.FS

var localTemplate *template.Template

func getLocalTemplate() *template.Template {
	if localTemplate == nil {
		tpl, err := makeTemplate().ParseFS(embedFS, "templates/local/*.yaml")
		_panic(err)

		localTemplate = tpl
	}

	return localTemplate
}

var configTemplate *template.Template

func getConfigTemplate() *template.Template {
	if configTemplate == nil {
		templateRoot := "templates/config"
		templatePattern := regexp.MustCompile(`\.yaml$`)

		tpl := makeTemplate()

		// We don't use tpl.ParseFS here to keep the folder structure in the template name.
		err := fs.WalkDir(embedFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Println(err)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if !templatePattern.MatchString(path) {
				return nil
			}

			name := strings.Replace(path, templateRoot+"/", "", 1)
			content, err := fs.ReadFile(embedFS, path)
			if err != nil {
				return err
			}
			_, err = tpl.New(name).Parse(string(content))
			if err != nil {
				return err
			}

			return nil
		})
		_panic(err)

		configTemplate = tpl
	}

	return configTemplate
}

func writeConfigDir(filesystem billy.Filesystem, gitRoot string) error {
	keConfig := getKeConfig()

	for _, tpl := range getConfigTemplate().Templates() {
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
