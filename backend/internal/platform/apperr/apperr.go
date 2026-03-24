package apperr

import "errors"

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string {
	return e.Message
}

func New(code, message string, status int) *Error {
	return &Error{Code: code, Message: message, Status: status}
}

func BadRequest(code, message string) *Error {
	return New(code, message, 400)
}

func Unauthorized(code, message string) *Error {
	return New(code, message, 401)
}

func Forbidden(code, message string) *Error {
	return New(code, message, 403)
}

func NotFound(code, message string) *Error {
	return New(code, message, 404)
}

func Conflict(code, message string) *Error {
	return New(code, message, 409)
}

func BadGateway(code, message string) *Error {
	return New(code, message, 502)
}

func Internal(code, message string) *Error {
	return New(code, message, 500)
}

func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("INTERNAL_ERROR", err.Error())
}
