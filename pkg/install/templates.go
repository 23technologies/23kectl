package install

import (
	"embed"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

var templateDir = "templates"
var distDir = "dist"

func newDomainConfigAzure(domain string) domainConfiguration {
	var dnsCredentials dnsCredentialsAzure
	dnsCredentials.parseCredentials()
	return domainConfiguration{
		Domain:      domain,
		Provider:    "azure-dns",
		Credentials: &dnsCredentials,
	}
}

func (d domainConfiguration) marshal() string {

	result, err := yaml.Marshal(d)
	if err != nil {
		panic("Error during marshaling")
	}

	return string(result)
}

func createDomainConfiguration(domain string, dnsProvider string) (domainConfiguration, error) {

	switch dnsProvider {
	case "azure-dns":
		return newDomainConfigAzure(domain), nil
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
		templatePattern := regexp.MustCompile("\\.yaml$")

		tpl := makeTemplate()

		// We don't use tpl.ParseFS here to keep the folder structure in the template name.
		fs.WalkDir(embedFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
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

		configTemplate = tpl
	}
	return configTemplate
}

func writeConfigDir(gitRoot string, keConfig *KeConfig) error {
	// todo: wipe gitRoot to account for deleted files

	for _, tpl := range getConfigTemplate().Templates() {
		name := tpl.Name()

		destPath := path.Join(gitRoot, name)
		destDir := path.Dir(destPath)

		err := os.MkdirAll(destDir, os.ModeDir|0700)
		if err != nil {
			return err
		}

		file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0600)
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
