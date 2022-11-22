package install

import "github.com/go-playground/validator/v10"

func makeValidator(tag string) func(value interface{}) error {
	vtor := validator.New()

	return func(value interface{}) error {
		return vtor.Var(value, tag)
	}
}
