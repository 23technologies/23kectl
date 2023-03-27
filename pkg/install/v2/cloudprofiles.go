package install

import (
	"fmt"
	"github.com/23technologies/23kectl/pkg/common"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"sort"
)

func GetAvailableCloudProfiles() ([]string, error) {
	const cloudprofileValuesPath = "helmcharts/cloudprofiles/values.yaml"

	bucket := viper.GetString("version")
	content, err := common.FetchObject(bucket, cloudprofileValuesPath)

	if err != nil {
		return nil, fmt.Errorf("couldn't fetch %s/%s: %w", bucket, cloudprofileValuesPath, err)
	}

	return ParseAvailableCloudProfiles(content)
}

func ParseAvailableCloudProfiles(yamlBytes []byte) ([]string, error) {
	mymap := make(map[string]any)
	result := []string{}

	err := yaml.Unmarshal(yamlBytes, &mymap)

	if err != nil {
		return nil, fmt.Errorf("couldn't parse available cloud profiles %w", err)
	}

	for key, value := range mymap {
		if mapValue, ok := value.(map[string]any); ok {
			if _, ok = mapValue["enabled"].(bool); ok {
				result = append(result, key)
			}
		}
	}

	sort.Strings(result)

	return result, nil
}
