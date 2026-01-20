// Package validator provides data validation functionality.
// Supports phone number and email format validation, supports phone number formats for multiple countries/regions.
package validator

import (
	// Standard library
	"regexp"
	"strings"
)

var (
	// phoneRegexCN matches Chinese mainland phone numbers
	phoneRegexCN = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// phoneRegexUS matches US phone numbers
	phoneRegexUS = regexp.MustCompile(`^\+?1[2-9]\d{2}[2-9]\d{6}$`)
	// phoneRegexUK matches UK phone numbers
	phoneRegexUK = regexp.MustCompile(`^\+?44[1-9]\d{8,9}$`)
	// phoneRegexInternational matches international phone numbers (general format, 7-15 digits, may include +)
	phoneRegexInternational = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
	// emailRegex stricter email validation (conforms to RFC 5322 standard)
	// Improvement: disallows consecutive dots, disallows starting or ending with dot, stricter domain part
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?@[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`)
)

// ValidatePhone validates phone number format (supports multiple countries/regions)
func ValidatePhone(phone string) bool {
	if phone == "" {
		return true // Allow empty
	}
	phone = strings.TrimSpace(phone)

	// Try to match various formats
	if phoneRegexCN.MatchString(phone) {
		return true // Chinese mainland
	}
	if phoneRegexUS.MatchString(phone) {
		return true // US
	}
	if phoneRegexUK.MatchString(phone) {
		return true // UK
	}
	// General international format (as fallback)
	return phoneRegexInternational.MatchString(phone)
}

// ValidateEmail validates email format (stricter validation)
func ValidateEmail(email string) bool {
	if email == "" {
		return true // Allow empty
	}
	email = strings.TrimSpace(email)

	// Basic format check
	if !emailRegex.MatchString(email) {
		return false
	}

	// Additional check: disallow consecutive dots
	if strings.Contains(email, "..") {
		return false
	}

	// Check parts before and after @ symbol
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	localPart := parts[0]
	domainPart := parts[1]

	// Local part cannot start or end with dot
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}

	// Domain part cannot start or end with dot
	if strings.HasPrefix(domainPart, ".") || strings.HasSuffix(domainPart, ".") {
		return false
	}

	// Domain part must contain at least one dot (for TLD)
	if !strings.Contains(domainPart, ".") {
		return false
	}

	return true
}

// ValidateUser validates user data
func ValidateUser(phone, email string) error {
	if !ValidatePhone(phone) {
		return &ValidationError{Field: "phone", Message: "无效的手机号格式"}
	}
	if !ValidateEmail(email) {
		return &ValidationError{Field: "email", Message: "无效的邮箱格式"}
	}
	return nil
}

// ValidationError validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
