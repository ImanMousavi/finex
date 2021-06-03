package helpers

import "github.com/gookit/validate"

type Errors struct {
	Errors []string `json:"errors"`
}

func (e Errors) Size() int {
	return len(e.Errors)
}

func Vaildate(payload interface{}, err_src *Errors) {
	v := validate.Struct(payload)
	if !v.Validate() {
		for _, errs := range v.Errors.All() {
			for _, err := range errs {
				err_src.Errors = append(err_src.Errors, err)
			}
		}
	}
}
