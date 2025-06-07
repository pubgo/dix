package dixinternal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pubgo/funk/errors"
)

// DixError 依赖注入相关错误
type DixError struct {
	Type    ErrorType
	Message string
	Details map[string]interface{}
	Cause   error
}

func (e *DixError) Error() string {
	var parts []string
	parts = append(parts, "["+string(e.Type)+"] "+e.Message)

	if len(e.Details) > 0 {
		var details []string
		for k, v := range e.Details {
			details = append(details, k+"="+fmt.Sprintf("%v", v))
		}
		parts = append(parts, "details: "+strings.Join(details, ", "))
	}

	if e.Cause != nil {
		parts = append(parts, "cause: "+e.Cause.Error())
	}

	return strings.Join(parts, "; ")
}

func (e *DixError) Unwrap() error {
	return e.Cause
}

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "VALIDATION"
	ErrorTypeProvider      ErrorType = "PROVIDER"
	ErrorTypeInjection     ErrorType = "INJECTION"
	ErrorTypeCyclicDep     ErrorType = "CYCLIC_DEPENDENCY"
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeInvocation    ErrorType = "INVOCATION"
	ErrorTypeConfiguration ErrorType = "CONFIGURATION"
)

// NewDixError 创建新的DixError
func NewDixError(errType ErrorType, message string) *DixError {
	return &DixError{
		Type:    errType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail 添加详细信息
func (e *DixError) WithDetail(key string, value interface{}) *DixError {
	e.Details[key] = value
	return e
}

// WithCause 添加原因错误
func (e *DixError) WithCause(cause error) *DixError {
	e.Cause = cause
	return e
}

// 预定义错误创建函数
func NewValidationError(message string) *DixError {
	return NewDixError(ErrorTypeValidation, message)
}

func NewProviderError(message string) *DixError {
	return NewDixError(ErrorTypeProvider, message)
}

func NewInjectionError(message string) *DixError {
	return NewDixError(ErrorTypeInjection, message)
}

func NewCyclicDependencyError(cycle []reflect.Type) *DixError {
	var typeNames []string
	for _, t := range cycle {
		typeNames = append(typeNames, t.String())
	}

	return NewDixError(ErrorTypeCyclicDep, "circular dependency detected").
		WithDetail("cycle_path", strings.Join(typeNames, " -> "))
}

func NewNotFoundError(typ reflect.Type) *DixError {
	return NewDixError(ErrorTypeNotFound, "provider not found").
		WithDetail("type", typ.String()).
		WithDetail("kind", typ.Kind().String())
}

func NewInvocationError(message string) *DixError {
	return NewDixError(ErrorTypeInvocation, message)
}

func NewConfigurationError(message string) *DixError {
	return NewDixError(ErrorTypeConfiguration, message)
}

// WrapError 包装现有错误
func WrapError(err error, errType ErrorType, message string) *DixError {
	err = errors.WrapCaller(err, 1)
	return NewDixError(errType, message).WithCause(err)
}

// IsErrorType 检查错误类型
func IsErrorType(err error, errType ErrorType) bool {
	if dixErr, ok := err.(*DixError); ok {
		return dixErr.Type == errType
	}
	return false
}

// GetErrorDetails 获取错误详情
func GetErrorDetails(err error) map[string]interface{} {
	if dixErr, ok := err.(*DixError); ok {
		return dixErr.Details
	}
	return nil
}
