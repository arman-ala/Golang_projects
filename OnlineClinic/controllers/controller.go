// Placeholder for controllers/controller.go
package controllers

import "regexp"

func validatePhoneNumber(phone string) bool {
	matched, _ := regexp.MatchString(`^09[0-9]{9}$`, phone)
	return matched
}

func validateNationalCode(code string) bool {
	matched, _ := regexp.MatchString(`^[0-9]{10}$`, code)
	return matched
}
