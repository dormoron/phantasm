package grpc

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/dormoron/eidola"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dormoron/phantasm/internal/endpoint"
	"github.com/dormoron/phantasm/middleware"
)

// GRPCServerOption 是gRPC服务器选项
type GRPCServerOption func(*GRPCServer)

// WithAddress 设置服务器地址
func WithAddress(addr string) GRPCServerOption {
	return func(s *GRPCServer) {
		s.addr = addr
	}
}

// WithTimeout 设置服务器超时
func WithTimeout(timeout time.Duration) GRPCServerOption {
	return func(s *GRPCServer) {
		s.timeout = timeout
	}
}

// WithMiddleware 设置gRPC中间件
func WithMiddleware(m ...middleware.Middleware) GRPCServerOption {
	return func(s *GRPCServer) {
		s.middleware = m
	}
}

// WithTLS 设置TLS配置
func WithTLS(cert, key string) GRPCServerOption {
	return func(s *GRPCServer) {
		s.tlsCert = cert
		s.tlsKey = key
	}
}

// WithGracefulStop 设置是否优雅停止
func WithGracefulStop(graceful bool) GRPCServerOption {
	return func(s *GRPCServer) {
		s.gracefulStop = graceful
	}
}

// WithName 设置服务名称
func WithName(name string) GRPCServerOption {
	return func(s *GRPCServer) {
		s.name = name
	}
}

// WithWeight 设置服务权重
func WithWeight(weight uint32) GRPCServerOption {
	return func(s *GRPCServer) {
		s.weight = weight
	}
}

// WithGroup 设置服务分组
func WithGroup(group string) GRPCServerOption {
	return func(s *GRPCServer) {
		s.group = group
	}
}

// GRPCServer 是gRPC服务器
type GRPCServer struct {
	name         string
	addr         string
	timeout      time.Duration
	middleware   []middleware.Middleware
	tlsCert      string
	tlsKey       string
	gracefulStop bool
	weight       uint32
	group        string
	server       *eidola.Server
}

// NewGRPCServer 创建一个新的gRPC服务器
func NewGRPCServer(opts ...GRPCServerOption) (*GRPCServer, error) {
	s := &GRPCServer{
		addr:         ":9000",
		timeout:      time.Second * 30,
		gracefulStop: true,
		name:         "phantasm-grpc-service",
	}
	for _, o := range opts {
		o(s)
	}

	// 创建TLS凭证
	var tlsCreds credentials.TransportCredentials
	if s.tlsCert != "" && s.tlsKey != "" {
		var err error
		tlsCreds, err = credentials.NewServerTLSFromFile(s.tlsCert, s.tlsKey)
		if err != nil {
			return nil, err
		}
	}

	// 创建eidola服务器
	serverOpts := []eidola.ServerOption{}

	// 设置TLS
	if tlsCreds != nil {
		serverOpts = append(serverOpts, eidola.ServerWithTLS(tlsCreds))
	}

	// 设置优雅关闭
	serverOpts = append(serverOpts, eidola.ServerWithGracefulStop(s.gracefulStop))

	// 设置权重
	serverOpts = append(serverOpts, eidola.ServerWithWeight(s.weight))

	// 设置分组
	if s.group != "" {
		serverOpts = append(serverOpts, eidola.ServerWithGroup(s.group))
	}

	// 创建服务器
	server, err := eidola.NewServer(s.name, serverOpts...)
	if err != nil {
		return nil, err
	}

	s.server = server
	return s, nil
}

// Start 启动gRPC服务器
func (s *GRPCServer) Start(ctx context.Context) error {
	return s.server.Start(s.addr)
}

// Stop 停止gRPC服务器
func (s *GRPCServer) Stop(ctx context.Context) error {
	return s.server.Close()
}

// Endpoint 返回gRPC服务器的URL端点
func (s *GRPCServer) Endpoint() (*url.URL, error) {
	addr := s.addr
	if addr[0] == ':' {
		addr = "0.0.0.0" + addr
	}
	schema := "grpc"
	if s.tlsCert != "" && s.tlsKey != "" {
		schema = "grpcs"
	}

	// 使用endpoint.NormalizeEndpoint处理URL
	normalized, err := endpoint.NormalizeEndpoint(addr, schema)
	if err != nil {
		return nil, err
	}

	return url.Parse(normalized)
}

// RegisterService 注册gRPC服务
func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.Server.RegisterService(desc, impl)
}

// Server 返回底层gRPC服务器
func (s *GRPCServer) Server() *grpc.Server {
	return s.server.Server
}

// Client 相关函数

// GRPCClientOption 是gRPC客户端选项
type GRPCClientOption func(*GRPCClientOptions)

// GRPCClientOptions 是gRPC客户端选项集合
type GRPCClientOptions struct {
	Endpoint     string
	Timeout      time.Duration
	TLSCert      string
	Insecure     bool
	Balancer     string
	DialOptions  []grpc.DialOption
	Interceptors []grpc.UnaryClientInterceptor
}

// WithEndpoint 设置客户端端点
func WithEndpoint(endpoint string) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.Endpoint = endpoint
	}
}

// WithClientTimeout 设置客户端超时
func WithClientTimeout(timeout time.Duration) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.Timeout = timeout
	}
}

// WithClientTLS 设置客户端TLS
func WithClientTLS(cert string) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.TLSCert = cert
	}
}

// WithInsecure 设置不安全连接
func WithInsecure() GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.Insecure = true
	}
}

// WithBalancer 设置负载均衡器
func WithBalancer(balancer string) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.Balancer = balancer
	}
}

// WithDialOption 添加拨号选项
func WithDialOption(opts ...grpc.DialOption) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.DialOptions = append(o.DialOptions, opts...)
	}
}

// WithClientInterceptor 添加客户端拦截器
func WithClientInterceptor(interceptors ...grpc.UnaryClientInterceptor) GRPCClientOption {
	return func(o *GRPCClientOptions) {
		o.Interceptors = append(o.Interceptors, interceptors...)
	}
}

// Dial 创建到gRPC服务器的连接
func Dial(ctx context.Context, opts ...GRPCClientOption) (*grpc.ClientConn, error) {
	options := &GRPCClientOptions{
		Timeout: time.Second * 10,
	}
	for _, o := range opts {
		o(options)
	}

	dialOpts := options.DialOptions

	// 设置超时
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 设置TLS
	if options.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else if options.TLSCert != "" {
		creds, err := credentials.NewClientTLSFromFile(options.TLSCert, "")
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	// 设置拦截器
	if len(options.Interceptors) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(options.Interceptors...))
	}

	// 设置负载均衡
	if options.Balancer != "" {
		dialOpts = append(dialOpts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, options.Balancer)))
	}

	return grpc.DialContext(ctx, options.Endpoint, dialOpts...)
}
