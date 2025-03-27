package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/middleware"
)

// UnaryServerInterceptor 创建一个gRPC一元拦截器，使用phantasm中间件
func UnaryServerInterceptor(m ...middleware.Middleware) grpc.UnaryServerInterceptor {
	chain := middleware.Chain(m...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 封装gRPC处理程序
		h := func(ctx context.Context, req interface{}) (interface{}, error) {
			return handler(ctx, req)
		}

		// 准备上下文
		ctx = setContextInfo(ctx, info.FullMethod)

		// 应用中间件链
		h = chain(h)

		// 执行处理程序
		resp, err := h(ctx, req)

		// 转换错误类型
		if err != nil {
			err = TranslateError(err)
		}

		return resp, err
	}
}

// StreamServerInterceptor 创建一个gRPC流拦截器，使用phantasm中间件
func StreamServerInterceptor(m ...middleware.Middleware) grpc.StreamServerInterceptor {
	chain := middleware.Chain(m...)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 封装流处理
		wrapper := &serverStreamWrapper{
			ServerStream: ss,
			ctx:          ss.Context(),
		}

		// 准备上下文
		wrapper.ctx = setContextInfo(wrapper.ctx, info.FullMethod)

		// 封装处理程序
		h := func(ctx context.Context, req interface{}) (interface{}, error) {
			err := handler(srv, wrapper)
			return nil, err
		}

		// 应用中间件链
		h = chain(h)

		// 执行处理程序（使用nil作为请求，因为流处理不需要）
		_, err := h(wrapper.ctx, nil)

		// 转换错误类型
		if err != nil {
			err = TranslateError(err)
		}

		return err
	}
}

// serverStreamWrapper 包装gRPC服务流，允许修改上下文
type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 重写Context方法，返回我们修改后的上下文
func (s *serverStreamWrapper) Context() context.Context {
	return s.ctx
}

// TranslateError 将phantasm错误转换为gRPC错误
func TranslateError(err error) error {
	if err == nil {
		return nil
	}

	// 如果已经是phantasm错误，转换为gRPC状态
	if e, ok := err.(*errors.Error); ok {
		return e
	}

	// 将普通错误转换为内部错误
	return errors.InternalServer("GRPC_ERROR", err.Error())
}

// setContextInfo 向上下文添加请求信息
func setContextInfo(ctx context.Context, fullMethod string) context.Context {
	// 添加方法信息
	ctx = context.WithValue(ctx, "method", fullMethod)
	ctx = context.WithValue(ctx, "path", fullMethod)

	// 添加客户端IP
	if p, ok := peer.FromContext(ctx); ok {
		ctx = context.WithValue(ctx, "client_ip", p.Addr.String())
	}

	// 添加请求ID
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// 尝试从元数据中获取请求ID
		if requestIDs := md.Get("x-request-id"); len(requestIDs) > 0 {
			ctx = context.WithValue(ctx, "request_id", requestIDs[0])
		}
	}

	return ctx
}

// WithUnaryServerInterceptors 添加gRPC一元拦截器选项
func WithUnaryServerInterceptors(interceptors ...grpc.UnaryServerInterceptor) ServerOption {
	return func(s *Server) {
		s.options = append(s.options, grpc.ChainUnaryInterceptor(interceptors...))
	}
}

// WithStreamServerInterceptors 添加gRPC流拦截器选项
func WithStreamServerInterceptors(interceptors ...grpc.StreamServerInterceptor) ServerOption {
	return func(s *Server) {
		s.options = append(s.options, grpc.ChainStreamInterceptor(interceptors...))
	}
}

// UseMiddleware 在gRPC服务器上使用phantasm中间件
func (s *Server) UseMiddleware(m ...middleware.Middleware) {
	s.options = append(s.options,
		grpc.ChainUnaryInterceptor(UnaryServerInterceptor(m...)),
		grpc.ChainStreamInterceptor(StreamServerInterceptor(m...)),
	)
}
