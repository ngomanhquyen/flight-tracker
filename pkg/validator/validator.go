// Package validator wraps go-playground/validator with the custom rules
// shared across services: IATA airport codes and airline flight numbers.
package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	iataPattern         = regexp.MustCompile(`^[A-Z]{3}$`)
	flightNumberPattern = regexp.MustCompile(`^[A-Z]{2,3}\d{1,4}[A-Z]?$`)
)

// New returns a *validator.Validate with the "iata" and "flightnumber"
// tags registered.
func New() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("iata", validateIATA)
	_ = v.RegisterValidation("flightnumber", validateFlightNumber)
	return v
}

func validateIATA(fl validator.FieldLevel) bool {
	return iataPattern.MatchString(strings.ToUpper(fl.Field().String()))
}

func validateFlightNumber(fl validator.FieldLevel) bool {
	return flightNumberPattern.MatchString(strings.ToUpper(fl.Field().String()))
}

// IsIATA reports whether s is a valid 3-letter IATA airport code
// (case-insensitive).
func IsIATA(s string) bool {
	return iataPattern.MatchString(strings.ToUpper(s))
}

// IsFlightNumber reports whether s is a valid flight number, e.g. VN257
// (case-insensitive).
func IsFlightNumber(s string) bool {
	return flightNumberPattern.MatchString(strings.ToUpper(s))
}
