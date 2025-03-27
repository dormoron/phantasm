package auth

import (
	"context"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是认证中间件的选项
type Option func(*options)

// Validator 是令牌验证器接口
type Validator interface {
	// Validate 验证认证信息并返回包含认证信息的上下文
	Validate(ctx context.Context) (context.Context, error)
}

// WithValidator 设置令牌验证器
func WithValidator(validator Validator) Option {
	return func(o *options) {
		o.validator = validator
	}
}

// options 是认证中间件的选项
type options struct {
	validator Validator
}

// Context 是身份验证上下文的键
type Context struct{}

// FromContext 从上下文中提取认证信息
func FromContext(ctx context.Context) (any, bool) {
	info, ok := ctx.Value(Context{}).(any)
	return info, ok
}

// Auth 返回一个认证中间件
func Auth(opts ...Option) middleware.Middleware {
	options := options{
		validator: defaultValidator{},
	}
	for _, o := range opts {
		o(&options)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			ctx, err := options.validator.Validate(ctx)
			if err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}
	}
}

// defaultValidator 是默认的令牌验证器，允许所有请求通过
type defaultValidator struct{}

// Validate 实现Validator接口
func (v defaultValidator) Validate(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// JWTValidator 是JWT验证器
type JWTValidator struct {
	Secret     string
	HeaderName string
}

// NewJWTValidator 创建一个新的JWT验证器
func NewJWTValidator(secret, headerName string) *JWTValidator {
	if headerName == "" {
		headerName = "Authorization"
	}
	return &JWTValidator{
		Secret:     secret,
		HeaderName: headerName,
	}
}

// Validate 实现Validator接口
func (v *JWTValidator) Validate(ctx context.Context) (context.Context, error) {
	// 从上下文中获取header
	token := getHeader(ctx, v.HeaderName)
	if token == "" {
		return ctx, errors.Unauthorized("AUTH_MISSING_TOKEN", "missing token")
	}

	// 解析bearer前缀
	const prefix = "Bearer "
	if len(token) <= len(prefix) || token[:len(prefix)] != prefix {
		return ctx, errors.Unauthorized("AUTH_INVALID_TOKEN", "invalid token format")
	}
	token = token[len(prefix):]

	// 这里应该实现JWT验证逻辑
	// 为简化示例，这里只判断token是否为空
	if token == "" {
		return ctx, errors.Unauthorized("AUTH_INVALID_TOKEN", "invalid token")
	}

	// 解析JWT并将认证信息添加到上下文
	// 示例中，我们只添加一个简单的用户ID
	ctx = context.WithValue(ctx, Context{}, map[string]interface{}{
		"user_id": "123", // 实际应用中，这应该从JWT中解析
	})

	return ctx, nil
}

// getHeader 从上下文中获取指定的HTTP头
func getHeader(ctx context.Context, name string) string {
	if headers, ok := ctx.Value("headers").(map[string]string); ok {
		return headers[name]
	}
	return ""
}
