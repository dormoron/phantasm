package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/dormoron/mist"

	"github.com/dormoron/phantasm/internal/endpoint"
	"github.com/dormoron/phantasm/internal/host"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/transport"
)

var _ transport.Server = (*Server)(nil)
var _ transport.Endpointer = (*Server)(nil)

// ServerOption 是HTTP服务器选项
type ServerOption func(*Server)

// Server 是HTTP服务器
type Server struct {
	*mist.HTTPServer
	server   *http.Server
	listener net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL
	network  string
	address  string
	timeout  time.Duration
	logger   log.Logger
}

// NewServer 创建HTTP服务器
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		HTTPServer: mist.InitHTTPServer(),
		network:    "tcp",
		address:    ":8000",
		timeout:    time.Second * 30,
		logger:     log.DefaultLogger,
	}
	for _, o := range opts {
		o(srv)
	}
	srv.server = &http.Server{
		Handler:      srv.HTTPServer,
		ReadTimeout:  srv.timeout,
		WriteTimeout: srv.timeout,
	}
	return srv
}

// Start 启动HTTP服务器
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}
	s.listener = listener

	// 使用internal/endpoint和host包构建端点URL
	hostname, port, err := host.ExtractHostPort(listener.Addr().String())
	if err != nil {
		return err
	}

	schema := "http"
	if s.tlsConf != nil {
		schema = "https"
	}

	// 构建host:port格式的地址
	addr := host.BuildAddress(hostname, port)
	s.endpoint = endpoint.NewEndpoint(schema, addr)

	s.logger.Info("[HTTP] server listening on: " + s.address)
	var serverErr error
	go func() {
		if s.tlsConf != nil {
			s.server.TLSConfig = s.tlsConf
			serverErr = s.server.ServeTLS(listener, "", "")
		} else {
			serverErr = s.server.Serve(listener)
		}
		if serverErr != nil && serverErr != http.ErrServerClosed {
			s.logger.Error("HTTP server error: " + serverErr.Error())
		}
	}()
	return nil
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("[HTTP] server stopping")
	return s.server.Shutdown(ctx)
}

// Endpoint 返回HTTP服务器的端点
func (s *Server) Endpoint() (*url.URL, error) {
	if s.listener == nil {
		return nil, errors.New("HTTP server is not started")
	}
	return s.endpoint, nil
}

// Network 设置网络类型，例如 "tcp", "tcp4", "tcp6", "unix" 或 "unixpacket"
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address 设置地址，例如 ":8000", "127.0.0.1:8000" 等
func Address(address string) ServerOption {
	return func(s *Server) {
		s.address = address
	}
}

// Timeout 设置超时时间
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Logger 设置日志记录器
func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// TLSConfig 设置TLS配置
func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

// SetHTTPServer 设置Mist HTTP服务器
func SetHTTPServer(server *mist.HTTPServer) ServerOption {
	return func(s *Server) {
		s.HTTPServer = server
	}
}
