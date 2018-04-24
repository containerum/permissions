package validation

import (
	"reflect"

	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v9"
)

type GinValidatorV9 struct {
	Validate *validator.Validate
}

var _ binding.StructValidator = &GinValidatorV9{}

func (v *GinValidatorV9) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		if err := v.Validate.Struct(obj); err != nil {
			return error(err)
		}
	}

	return nil
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}
