package proxy

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
)

type APIError struct {
	Reason  StatusReason
	Code    int
	Status  string
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

const (
	StatusSuccess = "Success"
	StatusFailure = "Failure"
)

type StatusReason string

const (
	StatusReasonUnknown StatusReason = "Unknown"

	// Status code 500
	StatusReasonInternalError StatusReason = "InternalError"

	// Status code 401
	StatusReasonUnauthorized StatusReason = "Unauthorized"

	// Status code 403
	StatusReasonForbidden StatusReason = "Forbidden"

	// Status code 404
	StatusReasonNotFound StatusReason = "NotFound"

	// Status code 409
	StatusReasonConflict StatusReason = "Conflict"

	// Status code 504
	StatusReasonGatewayTimeout StatusReason = "Timeout"

	// Status code 429
	StatusReasonTooManyRquests StatusReason = "TooManyRequests"

	// Status code 400
	StatusReasonBadRequest StatusReason = "BadRequest"

	// Status code 405
	StatusReasonMethodNotAllowed StatusReason = "MethodNotAllowed"

	// Status code 503
	StatusReasonServiceUnavailable StatusReason = "ServiceUnavailable"
)

func NewInternalServer(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonInternalError,
		Code:    http.StatusInternalServerError,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewUnauthorized(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonUnauthorized,
		Code:    http.StatusUnauthorized,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewForbidden(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonForbidden,
		Code:    http.StatusForbidden,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewNotFound(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonNotFound,
		Code:    http.StatusNotFound,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewConflict(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonConflict,
		Code:    http.StatusConflict,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewGatewayTimeout(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonGatewayTimeout,
		Code:    http.StatusGatewayTimeout,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewTooManyRequests(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonTooManyRquests,
		Code:    http.StatusTooManyRequests,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewBadRequest(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonBadRequest,
		Code:    http.StatusBadRequest,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewMethodNotAllowed(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonMethodNotAllowed,
		Code:    http.StatusMethodNotAllowed,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewServiceUnavailable(message string) *APIError {
	return &APIError{
		Reason:  StatusReasonServiceUnavailable,
		Code:    http.StatusServiceUnavailable,
		Status:  StatusFailure,
		Message: message,
	}
}

func NewUnknown(message string, statusCode int) *APIError {
	return &APIError{
		Reason:  StatusReasonUnknown,
		Code:    statusCode,
		Message: message,
		Status:  StatusFailure,
	}
}

func checkForAPIError(resp *http.Response) error {
	//HTTP status in 2xx range are success
	if c := resp.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBytes, err := ioutil.ReadAll(resp.Body)

	// Re-construct response body in case of error
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBytes))
	if err != nil {
		return err
	}

	message := string(respBytes)

	switch resp.StatusCode {

	case http.StatusInternalServerError:
		return NewInternalServer(message)

	case http.StatusUnauthorized:
		return NewUnauthorized(message)

	case http.StatusForbidden:
		return NewForbidden(message)

	case http.StatusNotFound:
		return NewNotFound(message)

	case http.StatusConflict:
		return NewConflict(message)

	case http.StatusGatewayTimeout:
		return NewGatewayTimeout(message)

	case http.StatusTooManyRequests:
		return NewTooManyRequests(message)

	case http.StatusMethodNotAllowed:
		return NewMethodNotAllowed(message)

	case http.StatusBadRequest:
		return NewBadRequest(message)

	case http.StatusServiceUnavailable:
		return NewServiceUnavailable(message)

	default:
		return NewUnknown(message, resp.StatusCode)
	}
}

func IsUnknown(err error) bool {
	return ReasonForError(err) == StatusReasonUnknown
}

func IsNotFound(err error) bool {
	return ReasonForError(err) == StatusReasonNotFound
}

func IsUnauthorized(err error) bool {
	return ReasonForError(err) == StatusReasonUnauthorized
}

func IsBadRequest(err error) bool {
	return ReasonForError(err) == StatusReasonBadRequest
}

func IsForbidden(err error) bool {
	return ReasonForError(err) == StatusReasonForbidden
}

func IsInternalServerError(err error) bool {
	return ReasonForError(err) == StatusReasonInternalError
}

func IsServiceUnavailable(err error) bool {
	return ReasonForError(err) == StatusReasonServiceUnavailable
}

func IsMethodNotAllowed(err error) bool {
	return ReasonForError(err) == StatusReasonMethodNotAllowed
}

func IsTooManyRequests(err error) bool {
	return ReasonForError(err) == StatusReasonTooManyRquests
}

func IsGatewayTimeout(err error) bool {
	return ReasonForError(err) == StatusReasonGatewayTimeout
}

func IsConflict(err error) bool {
	return ReasonForError(err) == StatusReasonConflict
}

func ReasonForError(err error) StatusReason {
	var e *APIError

	if errors.As(err, &e) {
		return e.Reason
	}

	return StatusReasonUnknown
}
