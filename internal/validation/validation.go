package validation

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validator = validator.New()

func Fields(err error) map[string][]string {
	result := make(map[string][]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, field := range validationErrors {
			name := strings.ToLower(field.Field())
			message := field.Error()
			switch field.Tag() {
			case "required":
				message = "is required"
			case "email":
				message = "must be a valid email"
			case "min":
				message = "is too short"
			case "max":
				message = "is too long"
			case "oneof":
				message = "has an unsupported value"
			}
			result[name] = append(result[name], strings.Title(message))
		}
	}
	return result
}
