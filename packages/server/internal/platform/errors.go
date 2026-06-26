package platform

import "fmt"

const (
	ErrorInvalidProjectURL = "INVALID_PROJECT_URL"
	ErrorInvalidMR         = "INVALID_MERGE_REQUEST"
	ErrorUnauthorized      = "UNAUTHORIZED"
	ErrorNotFound          = "NOT_FOUND"
	ErrorMalformedResponse = "MALFORMED_RESPONSE"
	ErrorHTTP              = "HTTP_ERROR"
)

// Error is a structured platform adapter error.
type Error struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: %s (status %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func platformError(code string, message string, statusCode int) *Error {
	return &Error{Code: code, Message: message, StatusCode: statusCode}
}
