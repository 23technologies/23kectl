package install

import (
	"encoding/base64"
	"fmt"
	"strings"
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
