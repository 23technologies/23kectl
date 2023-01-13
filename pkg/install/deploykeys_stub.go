//go:build test

package install

import "github.com/go-git/go-git/v5/plumbing/transport/ssh"

func init() {
	blockUntilKeyCanRead = func(_ string, _ *ssh.PublicKeys, _ string) {}

	// we need to have a server to get host keys from while testing,
	// although we don't actually need them.
	getHostname = func(_ any) string { return "github.com" }
}
