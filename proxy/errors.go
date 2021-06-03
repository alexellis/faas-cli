package proxy

import (
	"fmt"
	"net/http"
)

type OpenFaaSError struct {
	StatusCode int
	Message    string
}

func (e OpenFaaSError) Error() string {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Sprintf("unauthorized access, run \"faas-cli login\" to setup authentication for this server - http status %d", e.StatusCode)
	case http.StatusNotFound:
		return fmt.Sprintf("resource not found - http status %d", e.StatusCode)
	case http.StatusInternalServerError:
		return fmt.Sprintf("openfaas encountered an internal server error - http status: %d", e.StatusCode)
	case http.StatusBadRequest:
		return fmt.Sprintf("bad request - http status: %d", e.StatusCode)
	default:
		return fmt.Sprintf("%s - http status: %d", e.Message, e.StatusCode)
	}
}

func NewOpenFaaSError(message string, statusCode int) error {
	return OpenFaaSError{
		StatusCode: statusCode,
		Message:    message,
	}
}
