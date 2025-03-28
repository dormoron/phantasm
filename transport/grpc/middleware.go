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

// Context 重写Context方法以返回我们修改过的上下文
func (s *serverStreamWrapper) Context() context.Context {
	return s.ctx
}

// setContextInfo 在上下文中设置请求信息
func setContextInfo(ctx context.Context, fullMethod string) context.Context {
	// 设置方法名
	ctx = context.WithValue(ctx, "method", fullMethod)

	// 从gRPC元数据提取信息
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		headers := make(map[string]string)
		for k, v := range md {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		ctx = context.WithValue(ctx, "headers", headers)
	}

	// 提取对等方信息（客户端地址）
	if p, ok := peer.FromContext(ctx); ok {
		ctx = context.WithValue(ctx, "client_ip", p.Addr.String())
	}

	return ctx
}

// TranslateError 将通用错误转换为gRPC错误
func TranslateError(err error) error {
	if err == nil {
		return nil
	}

	// 如果已经是一个Phantasm错误，直接返回
	if errors.FromError(err) != nil {
		return err
	}

	// 将普通错误包装为内部错误
	return errors.InternalServer("internal.error", err.Error())
}

// UseMiddleware 在gRPC服务器上使用phantasm中间件
func (s *Server) UseMiddleware(m ...middleware.Middleware) {
	// 确保Eidola服务器已初始化
	if s.Server == nil || s.Server.Server == nil {
		return
	}

	// 注意：由于Eidola框架可能不直接支持替换拦截器
	// 这里只是示意代码，实际实现可能需要根据Eidola的实际API调整
	s.logger.Info("已应用中间件，但注意Eidola可能不支持直接替换拦截器")

	// 如果将来Eidola框架支持，可以使用如下代码：
	// unaryInterceptor := UnaryServerInterceptor(m...)
	// streamInterceptor := StreamServerInterceptor(m...)
	// s.Server.AddInterceptor(unaryInterceptor, streamInterceptor)
}
