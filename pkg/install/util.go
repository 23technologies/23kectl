package install

import (
	"fmt"
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
