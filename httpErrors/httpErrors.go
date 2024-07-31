package httpErrors

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	status  int
	message string
}

func (e *HttpError) Status() int {
	return e.status
}

func (e *HttpError) Message() string {
	return e.message
}

func (e *HttpError) Error() string {
	msg := fmt.Sprintf("HttpError_(%d)", e.status)

	if e.message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.message)
	}

	return msg
}

func (e *HttpError) WithMessage(msg string) error {
	e.message = msg
	return e
}

func NewHttpError(status int) *HttpError {
	return &HttpError{
		status:  status,
		message: "",
	}
}

func BadRequest() *HttpError {
	return NewHttpError(http.StatusBadRequest)
}

func Unauthorized() *HttpError {
	return NewHttpError(http.StatusUnauthorized)
}

func PaymentRequired() *HttpError {
	return NewHttpError(http.StatusPaymentRequired)
}

func Forbidden() *HttpError {
	return NewHttpError(http.StatusForbidden)
}

func NotFound() *HttpError {
	return NewHttpError(http.StatusNotFound)
}

func MethodNotAllowed() *HttpError {
	return NewHttpError(http.StatusMethodNotAllowed)
}

func NotAcceptable() *HttpError {
	return NewHttpError(http.StatusNotAcceptable)
}

func ProxyAuthenticationRequired() *HttpError {
	return NewHttpError(http.StatusProxyAuthRequired)
}

func RequestTimeout() *HttpError {
	return NewHttpError(http.StatusRequestTimeout)
}

func Conflict() *HttpError {
	return NewHttpError(http.StatusConflict)
}

func Gone() *HttpError {
	return NewHttpError(http.StatusGone)
}

func LengthRequired() *HttpError {
	return NewHttpError(http.StatusLengthRequired)
}

func PreconditionFailed() *HttpError {
	return NewHttpError(http.StatusPreconditionFailed)
}

func PayloadTooLarge() *HttpError {
	return NewHttpError(http.StatusRequestEntityTooLarge)
}

func UriTooLong() *HttpError {
	return NewHttpError(http.StatusRequestURITooLong)
}

func UnsupportedMediaType() *HttpError {
	return NewHttpError(http.StatusUnsupportedMediaType)
}

func RangeNotSatisfiable() *HttpError {
	return NewHttpError(http.StatusRequestedRangeNotSatisfiable)
}

func ExpectationFailed() *HttpError {
	return NewHttpError(http.StatusExpectationFailed)
}

func NotATeapot() *HttpError {
	return NewHttpError(418)
}

func MisdirectedRequest() *HttpError {
	return NewHttpError(http.StatusMisdirectedRequest)
}

func UnprocessableEntity() *HttpError {
	return NewHttpError(http.StatusUnprocessableEntity)
}

func Locked() *HttpError {
	return NewHttpError(http.StatusLocked)
}

func FailedDependency() *HttpError {
	return NewHttpError(http.StatusFailedDependency)
}

func TooEarly() *HttpError {
	return NewHttpError(http.StatusTooEarly)
}

func UpgradeRequired() *HttpError {
	return NewHttpError(http.StatusUpgradeRequired)
}

func PreconditionRequired() *HttpError {
	return NewHttpError(http.StatusPreconditionRequired)
}

func TooManyRequests() *HttpError {
	return NewHttpError(http.StatusTooManyRequests)
}

func RequestHeaderFieldsTooLarge() *HttpError {
	return NewHttpError(http.StatusRequestHeaderFieldsTooLarge)
}

func UnavailableForLegalReasons() *HttpError {
	return NewHttpError(http.StatusUnavailableForLegalReasons)
}

func InternalServerError() *HttpError {
	return NewHttpError(http.StatusInternalServerError)
}
