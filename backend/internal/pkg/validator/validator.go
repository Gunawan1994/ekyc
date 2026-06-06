package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator and implements echo.Validator.
type Validator struct {
	validator *validator.Validate
}

// New returns a ready-to-use Validator.
func New() *Validator {
	return &Validator{validator: validator.New()}
}

// Validate satisfies the echo.Validator interface.
// It returns nil on success, or a descriptive error listing every failed field.
func (v *Validator) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		ve, ok := err.(validator.ValidationErrors)
		if ok {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, fmt.Sprintf(
					"field '%s' failed on the '%s' rule",
					fe.Field(), fe.Tag(),
				))
			}
			return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
		}
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
