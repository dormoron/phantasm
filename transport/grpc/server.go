package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/dormoron/eidola"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/dormoron/phantasm/internal/endpoint"
	"github.com/dormoron/phantasm/internal/host"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/transport"
)

var _ transport.Server = (*Server)(nil)
var _ transport.Endpointer = (*Server)(nil)

// ServerOption 是gRPC服务器选项
type ServerOption func(*Server)

// Server 是gRPC服务器
type Server struct {
	*eidola.Server
	server   *grpc.Server
	listener net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL
	network  string
	address  string
	timeout  time.Duration
	logger   log.Logger
	health   *health.Server
	options  []grpc.ServerOption
	name     string // 服务名称
}

// NewServer 创建gRPC服务器
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		address: ":9000",
		timeout: time.Second * 30,
		logger:  log.DefaultLogger,
		health:  health.NewServer(),
		options: []grpc.ServerOption{},
		name:    "phantasm-grpc-service", // 默认服务名
	}
	for _, o := range opts {
		o(srv)
	}

	// 创建eidola服务器
	server, err := eidola.NewServer(srv.name)
	if err != nil {
		// 记录错误并返回默认服务器
		srv.logger.Error("Failed to create eidola server: " + err.Error())
	} else {
		srv.Server = server
	}

	// 创建grpc服务器
	grpcOpts := []grpc.ServerOption{}

	// 如果eidola服务器创建成功，添加拦截器
	if srv.Server != nil && srv.Server.Server != nil {
		// 注意：由于 eidola v0.1.0 版本可能不支持这些拦截器方法，暂时不添加
		// 未来版本可以取消注释启用
		/*
			grpcOpts = append(grpcOpts,
				grpc.ChainUnaryInterceptor(
					srv.Server.UnaryServerInterceptor(),
				),
				grpc.ChainStreamInterceptor(
					srv.Server.StreamServerInterceptor(),
				),
			)
		*/
	}

	if srv.tlsConf != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(srv.tlsConf)))
	}

	// 添加自定义选项
	grpcOpts = append(grpcOpts, srv.options...)

	srv.server = grpc.NewServer(grpcOpts...)

	// 注册健康检查
	grpc_health_v1.RegisterHealthServer(srv.server, srv.health)
	// 注册反射服务，以支持grpcurl等工具
	reflection.Register(srv.server)
	return srv
}

// Start 启动gRPC服务器
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

	schema := "grpc"
	if s.tlsConf != nil {
		schema = "grpcs"
	}

	// 构建host:port格式的地址
	addr := host.BuildAddress(hostname, port)
	s.endpoint = endpoint.NewEndpoint(schema, addr)

	s.logger.Info("[gRPC] server listening on: " + s.address)
	s.health.Resume()
	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.logger.Error("[gRPC] serve error: " + err.Error())
		}
	}()
	return nil
}

// Stop 停止gRPC服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("[gRPC] server stopping")
	s.health.Shutdown()
	s.server.GracefulStop()
	return nil
}

// Endpoint 返回gRPC服务器的端点
func (s *Server) Endpoint() (*url.URL, error) {
	if s.listener == nil {
		return nil, errors.New("gRPC server is not started")
	}
	return s.endpoint, nil
}

// RegisterService 注册gRPC服务
func (s *Server) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	s.server.RegisterService(sd, ss)
}

// Network 设置网络类型，例如 "tcp", "tcp4", "tcp6", "unix" 或 "unixpacket"
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address 设置地址，例如 ":9000", "127.0.0.1:9000" 等
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

// Options 添加grpc服务器选项
func Options(opts ...grpc.ServerOption) ServerOption {
	return func(s *Server) {
		s.options = append(s.options, opts...)
	}
}

// EidolaServer 设置Eidola服务器
func EidolaServer(server *eidola.Server) ServerOption {
	return func(s *Server) {
		s.Server = server
	}
}

// Name 设置服务名称
func Name(name string) ServerOption {
	return func(s *Server) {
		s.name = name
	}
}
