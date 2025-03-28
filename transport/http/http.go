package http

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/dormoron/mist"

	"github.com/dormoron/phantasm/internal/endpoint"
	"github.com/dormoron/phantasm/middleware"
)

// HTTPServerOption 是HTTP服务器选项
type HTTPServerOption func(*HTTPServer)

// WithAddress 设置服务器地址
func WithAddress(addr string) HTTPServerOption {
	return func(s *HTTPServer) {
		s.addr = addr
	}
}

// WithTimeout 设置服务器超时
func WithTimeout(timeout time.Duration) HTTPServerOption {
	return func(s *HTTPServer) {
		s.timeout = timeout
	}
}

// WithTLS 设置TLS配置
func WithTLS(cert, key string) HTTPServerOption {
	return func(s *HTTPServer) {
		s.tlsCert = cert
		s.tlsKey = key
	}
}

// HTTPServer 是HTTP服务器
type HTTPServer struct {
	addr       string
	timeout    time.Duration
	middleware []middleware.Middleware
	tlsCert    string
	tlsKey     string
	mistServer *mist.HTTPServer
	running    bool
	endpoint   *url.URL
	httpServer *http.Server // 添加标准库的HTTP服务器实例，用于优雅关闭
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(opts ...HTTPServerOption) (*HTTPServer, error) {
	// 创建mist HTTP服务器选项
	mistOpts := []mist.HTTPServerOption{}

	s := &HTTPServer{
		addr:       ":8000",
		timeout:    time.Second * 30,
		mistServer: mist.InitHTTPServer(mistOpts...),
		running:    false,
	}

	for _, o := range opts {
		o(s)
	}

	// 应用初始化时设置的中间件（兼容性支持）
	if len(s.middleware) > 0 {
		s.UseMiddleware(s.middleware...)
	}

	return s, nil
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start(ctx context.Context) error {
	// 构建端点URL
	scheme := "http"
	if s.tlsCert != "" && s.tlsKey != "" {
		scheme = "https"
	}

	host := s.addr
	if host[0] == ':' {
		host = "0.0.0.0" + host
	}

	normalized, err := endpoint.NormalizeEndpoint(host, scheme)
	if err != nil {
		return err
	}

	s.endpoint, err = url.Parse(normalized)
	if err != nil {
		return err
	}

	// 创建一个标准库的HTTP服务器，封装mist服务器
	httpServer := &http.Server{
		Addr:         s.addr,
		Handler:      s.mistServer,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.timeout,
	}

	// 保存标准HTTP服务器实例，用于后续的优雅关闭
	s.httpServer = httpServer

	// 启动服务器
	go func() {
		var err error
		if s.tlsCert != "" && s.tlsKey != "" {
			// 使用标准库的ListenAndServeTLS进行TLS支持
			err = httpServer.ListenAndServeTLS(s.tlsCert, s.tlsKey)
		} else {
			// 为了统一处理，我们使用标准库的http.Server
			// 而不是直接调用s.mistServer.Start(s.addr)
			err = httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			// 如果有需要，这里可以添加错误日志
		}
	}()

	s.running = true
	return nil
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.running = false

	// 如果我们有标准库的HTTP服务器实例，使用其优雅关闭功能
	if s.httpServer != nil {
		// 创建一个子上下文用于关闭超时控制
		shutdownCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		// 使用标准库的Shutdown方法进行优雅关闭
		return s.httpServer.Shutdown(shutdownCtx)
	}

	// 退化情况：如果没有httpServer实例，等待超时
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.timeout):
		return nil
	}
}

// Endpoint 返回HTTP服务器的URL端点
func (s *HTTPServer) Endpoint() (*url.URL, error) {
	if !s.running {
		return nil, errors.New("HTTP server is not started")
	}
	return s.endpoint, nil
}

// Group 创建路由组
func (s *HTTPServer) Group(prefix string) interface{} {
	return s.mistServer.Group(prefix)
}

// UseMiddleware 在HTTP服务器上使用Phantasm中间件
func (s *HTTPServer) UseMiddleware(middleware ...middleware.Middleware) {
	for _, m := range middleware {
		// 将Phantasm中间件适配到Mist中间件
		s.mistServer.Use(func(next mist.HandleFunc) mist.HandleFunc {
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
					// 处理错误（后续可以扩展具体错误处理逻辑）
					c.AbortWithStatus(500)
					c.RespondWithJSON(500, map[string]string{
						"error": err.Error(),
					})
				}
			}
		})
	}
}

// GET 注册GET方法处理程序
func (s *HTTPServer) GET(path string, handler mist.HandleFunc) {
	s.mistServer.GET(path, handler)
}

// POST 注册POST方法处理程序
func (s *HTTPServer) POST(path string, handler mist.HandleFunc) {
	s.mistServer.POST(path, handler)
}

// PUT 注册PUT方法处理程序
func (s *HTTPServer) PUT(path string, handler mist.HandleFunc) {
	s.mistServer.PUT(path, handler)
}

// DELETE 注册DELETE方法处理程序
func (s *HTTPServer) DELETE(path string, handler mist.HandleFunc) {
	s.mistServer.DELETE(path, handler)
}

// GetEngine 返回内部的mist HTTP服务器引擎
func (s *HTTPServer) GetEngine() *mist.HTTPServer {
	return s.mistServer
}
