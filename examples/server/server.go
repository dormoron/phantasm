package server

import (
	"context"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware/logging"
	"github.com/dormoron/phantasm/middleware/recovery"
	"github.com/dormoron/phantasm/transport/grpc"
	"github.com/dormoron/phantasm/transport/http"
)

// 定义示例服务接口
type ExampleService interface {
	SayHello(name string) string
}

// 示例服务实现
type exampleServiceImpl struct{}

func (s *exampleServiceImpl) SayHello(name string) string {
	return "Hello, " + name + "!"
}

// NewHTTPServer 创建基于Mist的HTTP服务器
func NewHTTPServer(addr string, logger log.Logger, svc ExampleService) (*http.HTTPServer, error) {
	// 创建HTTP服务器
	server, err := http.NewHTTPServer(
		http.WithAddress(addr),
		http.WithTimeout(time.Second*5),
	)
	if err != nil {
		return nil, err
	}

	// 使用中间件
	server.UseMiddleware(
		recovery.Recovery(),
		logging.Logging(
			logging.WithLogRequestBody(true),
			logging.WithLogResponseBody(true),
		),
	)

	// 注册API路由
	mistEngine := server.GetEngine()

	// 添加路由处理程序
	mistEngine.GET("/api/v1/hello/:name", func(c *mist.Context) {
		nameVal, err := c.PathValue("name").String()
		if err != nil {
			c.RespondWithJSON(400, map[string]string{"error": "无效的名称参数"})
			return
		}
		message := svc.SayHello(nameVal)
		c.RespondWithJSON(200, map[string]interface{}{
			"message": message,
		})
	})

	// 健康检查路由
	mistEngine.GET("/health", func(c *mist.Context) {
		c.RespondWithJSON(200, map[string]string{
			"status": "ok",
		})
	})

	return server, nil
}

// NewGRPCServer 创建基于Eidola的gRPC服务器
func NewGRPCServer(addr string, logger log.Logger) (*grpc.Server, error) {
	// 创建gRPC服务器
	server := grpc.NewServer(
		grpc.Address(addr),
		grpc.Timeout(time.Second*5),
		grpc.Logger(logger),
		grpc.Name("example-service"),
	)

	// 使用中间件
	server.UseMiddleware(
		recovery.Recovery(),
		logging.Logging(
			logging.WithLogRequestBody(true),
			logging.WithLogResponseBody(true),
		),
	)

	// 注意：这里应该注册gRPC服务定义
	// 例如：pb.RegisterExampleServiceServer(server, &exampleServiceServer{})
	// 但由于这只是示例，我们省略了具体的protobuf生成代码部分

	return server, nil
}

// StartServers 启动HTTP和gRPC服务器
func StartServers(ctx context.Context, logger log.Logger) error {
	// 创建服务实例
	svc := &exampleServiceImpl{}

	// 创建HTTP服务器
	httpServer, err := NewHTTPServer(":8000", logger, svc)
	if err != nil {
		return err
	}

	// 创建gRPC服务器
	grpcServer, err := NewGRPCServer(":9000", logger)
	if err != nil {
		return err
	}

	// 启动HTTP服务器
	if err := httpServer.Start(ctx); err != nil {
		return err
	}
	logger.Info("HTTP server started on :8000")

	// 启动gRPC服务器
	if err := grpcServer.Start(ctx); err != nil {
		return err
	}
	logger.Info("gRPC server started on :9000")

	// 返回nil表示启动成功
	return nil
}
