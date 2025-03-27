package middleware

import "context"

// Handler 定义请求处理程序
type Handler func(ctx context.Context, req interface{}) (interface{}, error)

// Middleware 是中间件函数
type Middleware func(Handler) Handler

// Chain 将多个中间件链接在一起
func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}
