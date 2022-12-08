package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/Masterminds/semver/v3"
	"github.com/akrennmair/slice"
)

func Panic(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func PressEnterToContinue() {
	fmt.Println("Press the Enter Key to continue")
	fmt.Scanln()
}

func CoerceBase64String(s string) string {
	if isBase64String(s) {
		return s
	} else {
		return base64String(s)
	}
}

func isBase64String(s string) bool {
	err := validator.New().Var(s, "base64")
	return err == nil
}

func base64String(s string) string {
	bob := strings.Builder{}
	base64.NewEncoder(base64.StdEncoding, &bob).Write([]byte(s))

	return bob.String()
}

const colorErr = color.FgRed
const colorHighlight = color.FgBlue

// const colorSuccess = color.FgGreen
const colorWarn = color.FgYellow

var PrintErr = color.New(colorErr).PrintlnFunc()
var PrintHighlight = color.New(colorHighlight).PrintlnFunc()

// var printSuccess = color.New(colorSuccess).PrintlnFunc()
var PrintWarn = color.New(colorWarn).PrintlnFunc()

// list23keTag ...
func List23keTags(publicKeys *ssh.PublicKeys) ([]string, error) {

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
				continue
			}
			versions = append(versions, v)
		}
	}

	sort.Sort(semver.Collection(versions))

	maxMinor := versions[len(versions)-1].Minor()
	var maxMinorMinus2 uint64
	if maxMinor <= 2 {
		maxMinorMinus2 = 0
	} else {
		maxMinorMinus2 = maxMinor - 2
	}

	versions = slice.Filter(versions, func(v *semver.Version) bool { return v.Minor() >= maxMinorMinus2 })
	// reverse the order of versions in order to list latest version first
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}
	return slice.Map(versions, func(v *semver.Version) string { return "v" + v.String() }), nil
}

func RandHex(bytes int) string {
	byteArr := make([]byte, bytes)
	rand.Read(byteArr)
	return hex.EncodeToString(byteArr)
}
