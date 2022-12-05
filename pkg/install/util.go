package install

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/Masterminds/semver/v3"
	"github.com/akrennmair/slice"
)

func _panic(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func pressEnterToContinue() {
	fmt.Println("Press the Enter Key to continue")
	fmt.Scanln()
}

func base64String(s string) string {
	bob := strings.Builder{}
	base64.NewEncoder(base64.StdEncoding, &bob).Write([]byte(s))

	return bob.String()
}


const colorErr = color.FgRed
const colorHighlight = color.FgBlue
//const colorSuccess = color.FgGreen
const colorWarn = color.FgYellow

var printErr = color.New(colorErr).PrintlnFunc()
var printHighlight = color.New(colorHighlight).PrintlnFunc()
// var printSuccess = color.New(colorSuccess).PrintlnFunc()
var printWarn = color.New(colorWarn).PrintlnFunc()

// list23keTag ...
func list23keTags(publicKeys *ssh.PublicKeys) ([]string, error) {

	rem := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "23ke-origin",
		URLs: []string{"ssh://git@github.com/23technologies/23ke"},
	})

	refs, err := rem.List(&git.ListOptions{
		Auth: publicKeys,
	})
	if err != nil {
		return nil, err
	}

	// Filters the references list and only keeps tags
	var versions []*semver.Version
	for _, ref := range refs {
		if ref.Name().IsTag() {
			v, err := semver.NewVersion(string(ref.Name().Short()))
			if err != nil {
				return nil, err
			}
			versions = append(versions, v)
		}
	}

	sort.Sort(semver.Collection(versions))

	maxMinor := versions[len(versions)-1].Minor()
	var maxMinorMinus3 uint64
	if maxMinor <= 3 {
		maxMinorMinus3 = 0
	} else {
		maxMinorMinus3 = maxMinor - 3
	}

	versions = slice.Filter(versions, func(v *semver.Version) bool { return v.Minor() >= maxMinorMinus3 })
	return slice.Map(versions, func(v *semver.Version) string { return v.String() }), nil
}

func randHex(bytes int) string {
	byteArr := make([]byte, bytes)
	rand.Read(byteArr)
	return hex.EncodeToString(byteArr)
}
