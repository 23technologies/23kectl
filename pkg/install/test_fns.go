//go:build test

package install

import (
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/viper"
	"net/url"
	"strings"
)

var TestConfig map[string]any
var HostName = "github.com"

func init() {
	blockUntilKeyCanRead = func(_ string, _ *ssh.PublicKeys, _ string) {}

	// we need to have a server to get host keys from while testing,
	// although we don't actually need them.
	getSSHHostname = func(_ *url.URL) string { return HostName }

	queryConfigKey = func(configKey string, _ func() (any, error)) error {
		lc := strings.ToLower(configKey)

		result := TestConfig[lc]

		if result == "" {
			panic("key doesn't exist: " + lc)
		}

		viper.Set(configKey, result)

		return nil
	}
}
