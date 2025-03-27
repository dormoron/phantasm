package http

import (
	"context"
	"net/http"

	"github.com/dormoron/mist"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/middleware"
)

// MiddlewareAdapter 将phantasm中间件适配到mist中间件
func MiddlewareAdapter(m middleware.Middleware) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(c *mist.Context) {
			// 包装一个phantasm处理程序
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				// 将mist上下文作为请求的上下文
				c.Request = c.Request.WithContext(ctx)

				// 调用下一个处理程序
				next(c)

				// 返回空响应（因为响应已通过mist.Context直接写入）
				return nil, nil
			}

			// 准备上下文信息
			ctx := c.Request.Context()
			ctx = context.WithValue(ctx, "path", c.Request.URL.Path)
			ctx = context.WithValue(ctx, "method", c.Request.Method)

			// 添加头信息到上下文
			headers := make(map[string]string)
			for k, v := range c.Request.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			ctx = context.WithValue(ctx, "headers", headers)

			// 获取客户端IP
			clientIP := c.ClientIP()
			if clientIP != "" {
				ctx = context.WithValue(ctx, "client_ip", clientIP)
			}

			// 应用phantasm中间件
			adaptedHandler := m(handler)

			// 调用适配后的处理程序
			_, err := adaptedHandler(ctx, c.Request)
			if err != nil {
				// 处理错误
				statusCode := http.StatusInternalServerError

				// 尝试从phantasm错误中获取状态码
				if phantasmErr, ok := err.(*errors.Error); ok {
					statusCode = int(phantasmErr.Code)
				}

				// 使用正确的Context API设置状态码和响应
				c.AbortWithStatus(statusCode)
				c.RespondWithJSON(statusCode, map[string]string{
					"error": err.Error(),
				})
				return
			}
		}
	}
}

// UseMiddleware 在HTTP服务器上使用Cosmos中间件
func (s *HTTPServer) UseMiddleware(middleware ...middleware.Middleware) {
	for _, m := range middleware {
		s.mistServer.Use(MiddlewareAdapter(m))
	}
}

// UseMiddleware 在路由组上使用Cosmos中间件
func (g *RouterGroup) UseMiddleware(middleware ...middleware.Middleware) {
	for _, m := range middleware {
		g.mistServer.Use(MiddlewareAdapter(m))
	}
}
