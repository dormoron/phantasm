package http

import (
	"context"
	"crypto/tls"
	"net"
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

// WithMiddleware 设置HTTP中间件（已废弃，请使用UseMiddleware方法）
// Deprecated: 使用HTTPServer.UseMiddleware代替
func WithMiddleware(m ...middleware.Middleware) HTTPServerOption {
	return func(s *HTTPServer) {
		s.middleware = m
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
	server     *http.Server
	mistServer *mist.HTTPServer
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(opts ...HTTPServerOption) (*HTTPServer, error) {
	s := &HTTPServer{
		addr:       ":8000",
		timeout:    time.Second * 30,
		mistServer: mist.InitHTTPServer(),
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
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.mistServer,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.timeout,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	if s.tlsCert != "" && s.tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(s.tlsCert, s.tlsKey)
		if err != nil {
			return err
		}
		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		return s.server.ServeTLS(ln, s.tlsCert, s.tlsKey)
	}

	return s.server.Serve(ln)
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Endpoint 返回HTTP服务器的URL端点
func (s *HTTPServer) Endpoint() (*url.URL, error) {
	addr := s.addr
	if addr[0] == ':' {
		addr = "0.0.0.0" + addr
	}
	scheme := "http"
	if s.tlsCert != "" && s.tlsKey != "" {
		scheme = "https"
	}

	// 使用endpoint.NormalizeEndpoint处理URL
	normalized, err := endpoint.NormalizeEndpoint(addr, scheme)
	if err != nil {
		return nil, err
	}

	return url.Parse(normalized)
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

// Group 创建路由组
func (s *HTTPServer) Group(prefix string) *RouterGroup {
	if prefix == "" || prefix[0] != '/' {
		prefix = "/" + prefix
	}

	return &RouterGroup{
		prefix:     prefix,
		mistServer: s.mistServer,
	}
}

// RouterGroup 是路由组接口
type RouterGroup struct {
	prefix     string
	parent     *RouterGroup
	mistServer *mist.HTTPServer
}

// calculateFullPath 计算完整路径
func (g *RouterGroup) calculateFullPath(path string) string {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	if g.parent == nil {
		return g.prefix + path
	}

	return g.parent.calculateFullPath("") + g.prefix + path
}

// Group 创建子路由组
func (g *RouterGroup) Group(prefix string) *RouterGroup {
	if prefix == "" || prefix[0] != '/' {
		prefix = "/" + prefix
	}

	return &RouterGroup{
		prefix:     prefix,
		parent:     g,
		mistServer: g.mistServer,
	}
}

// GET 在路由组中注册GET方法处理程序
func (g *RouterGroup) GET(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.GET(fullPath, handler)
}

// POST 在路由组中注册POST方法处理程序
func (g *RouterGroup) POST(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.POST(fullPath, handler)
}

// PUT 在路由组中注册PUT方法处理程序
func (g *RouterGroup) PUT(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.PUT(fullPath, handler)
}

// DELETE 在路由组中注册DELETE方法处理程序
func (g *RouterGroup) DELETE(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.DELETE(fullPath, handler)
}

// OPTIONS 在路由组中注册OPTIONS方法处理程序
func (g *RouterGroup) OPTIONS(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.OPTIONS(fullPath, handler)
}

// HEAD 在路由组中注册HEAD方法处理程序
func (g *RouterGroup) HEAD(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.HEAD(fullPath, handler)
}

// PATCH 在路由组中注册PATCH方法处理程序
func (g *RouterGroup) PATCH(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.PATCH(fullPath, handler)
}

// Any 在路由组中注册处理任意HTTP方法的处理程序
func (g *RouterGroup) Any(path string, handler mist.HandleFunc) {
	fullPath := g.calculateFullPath(path)
	g.mistServer.GET(fullPath, handler)
	g.mistServer.POST(fullPath, handler)
	g.mistServer.PUT(fullPath, handler)
	g.mistServer.DELETE(fullPath, handler)
	g.mistServer.OPTIONS(fullPath, handler)
	g.mistServer.HEAD(fullPath, handler)
	g.mistServer.PATCH(fullPath, handler)
}

// Use 应用中间件到路由组
func (g *RouterGroup) Use(middlewares ...mist.Middleware) {
	for _, middleware := range middlewares {
		g.mistServer.Use(middleware)
	}
}
