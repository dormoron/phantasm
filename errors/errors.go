package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error 是Cosmos框架的错误类型
type Error struct {
	Code     int32             `json:"code"`
	Reason   string            `json:"reason"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("error: code = %d reason = %s message = %s metadata = %v", e.Code, e.Reason, e.Message, e.Metadata)
}

// New 创建一个新的错误
func New(code int32, reason, message string) *Error {
	return &Error{
		Code:     code,
		Reason:   reason,
		Message:  message,
		Metadata: make(map[string]string),
	}
}

// FromError 从error中创建Error
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return se
	}
	return New(http.StatusInternalServerError, "UnknownError", err.Error())
}

// Is 报告目标错误是否与此错误匹配
func (e *Error) Is(err error) bool {
	if se := new(Error); errors.As(err, &se) {
		return se.Reason == e.Reason
	}
	return false
}

// WithMetadata 向错误添加元数据
func (e *Error) WithMetadata(md map[string]string) *Error {
	e.Metadata = md
	return e
}

// WithMessage 更改错误消息
func (e *Error) WithMessage(message string) *Error {
	e.Message = message
	return e
}

// BadRequest 返回HTTP 400错误
func BadRequest(reason, message string) *Error {
	return New(http.StatusBadRequest, reason, message)
}

// Unauthorized 返回HTTP 401错误
func Unauthorized(reason, message string) *Error {
	return New(http.StatusUnauthorized, reason, message)
}

// Forbidden 返回HTTP 403错误
func Forbidden(reason, message string) *Error {
	return New(http.StatusForbidden, reason, message)
}

// NotFound 返回HTTP 404错误
func NotFound(reason, message string) *Error {
	return New(http.StatusNotFound, reason, message)
}

// Conflict 返回HTTP 409错误
func Conflict(reason, message string) *Error {
	return New(http.StatusConflict, reason, message)
}

// TooManyRequests 返回HTTP 429错误
func TooManyRequests(reason, message string) *Error {
	return New(http.StatusTooManyRequests, reason, message)
}

// InternalServer 返回HTTP 500错误
func InternalServer(reason, message string) *Error {
	return New(http.StatusInternalServerError, reason, message)
}

// ServiceUnavailable 返回HTTP 503错误
func ServiceUnavailable(reason, message string) *Error {
	return New(http.StatusServiceUnavailable, reason, message)
}

// IsNotFound 检查是否为NotFound错误
func IsNotFound(err error) bool {
	if se := FromError(err); se != nil {
		return se.Code == http.StatusNotFound
	}
	return false
}

// IsBadRequest 检查是否为BadRequest错误
func IsBadRequest(err error) bool {
	if se := FromError(err); se != nil {
		return se.Code == http.StatusBadRequest
	}
	return false
}

// IsInternalServer 检查是否为InternalServer错误
func IsInternalServer(err error) bool {
	if se := FromError(err); se != nil {
		return se.Code == http.StatusInternalServerError
	}
	return false
}
