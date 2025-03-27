# Cosmos 中间件

Cosmos框架的中间件模块提供了一系列通用中间件，用于处理常见的Web服务需求。

## 可用中间件

Cosmos框架提供以下中间件：

1. **logging**: 请求日志记录
2. **recovery**: 异常恢复
3. **auth**: 身份验证
4. **ratelimit**: 速率限制
5. **metrics**: 指标收集
6. **tracing**: 分布式跟踪
7. **observability**: 可观测性组合（日志+指标+跟踪）

## 使用方法

### 在HTTP服务中使用中间件

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/dormoron/mist"
	
	"github.com/dormoron/phantasm/middleware/auth"
	"github.com/dormoron/phantasm/middleware/logging"
	"github.com/dormoron/phantasm/middleware/recovery"
	"github.com/dormoron/phantasm/middleware/ratelimit"
	"github.com/dormoron/phantasm/transport/http"
)

func main() {
	// 创建HTTP服务器
	server, err := http.NewHTTPServer(
		http.WithAddress(":8000"),
		http.WithTimeout(time.Second * 30),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 使用中间件
	server.UseMiddleware(
		recovery.Recovery(),
		logging.Logging(
			logging.WithLogRequestBody(true),
			logging.WithLogResponseBody(true),
			logging.WithMaxBodyLogSize(8192), // 限制日志体大小为8KB
		),
		auth.Auth(
			auth.WithValidator(auth.NewJWTValidator("your-secret", "")),
		),
		ratelimit.RateLimit(
			ratelimit.WithLimiter(ratelimit.NewIPRateLimiter(100, 1000)),
		),
	)

	// 注册路由
	server.GET("/", func(c *mist.Context) {
		c.RespondWithJSON(200, map[string]string{"message": "Hello, World!"})
	})

	// 启动服务器
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

### 在gRPC服务中使用中间件

```go
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "your-project/api/helloworld"

	"github.com/dormoron/phantasm/middleware/auth"
	"github.com/dormoron/phantasm/middleware/logging"
	"github.com/dormoron/phantasm/middleware/recovery"
	"github.com/dormoron/phantasm/middleware/tracing"
	"github.com/dormoron/phantasm/transport/grpc"
)

// server 是gRPC服务器实现
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello 实现gRPC方法
func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func main() {
	// 创建gRPC服务器
	srv := grpc.NewServer(
		grpc.Address(":9000"),
		grpc.Timeout(time.Second * 30),
	)

	// 使用中间件
	srv.UseMiddleware(
		recovery.Recovery(),
		logging.Logging(
			logging.WithLogRequestBody(true),
			logging.WithLogResponseBody(true),
		),
		tracing.Server(),
		auth.Auth(),
	)

	// 注册gRPC服务
	pb.RegisterGreeterServer(srv.server, &server{})

	// 启动服务器
	if err := srv.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	// 阻塞直到服务器停止
	select {}
}
```

### 使用可观测性中间件

可观测性中间件整合了日志、指标和跟踪功能，提供了完整的请求观测能力：

```go
package main

import (
	"context"
	"log"

	"github.com/dormoron/phantasm/middleware/auth"
	"github.com/dormoron/phantasm/middleware/observability"
	"github.com/dormoron/phantasm/middleware/recovery"
	"github.com/dormoron/phantasm/transport/http"
)

func main() {
	// 创建HTTP服务器
	server, err := http.NewHTTPServer(
		http.WithAddress(":8000"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 使用可观测性中间件（整合日志、指标和跟踪）
	server.UseMiddleware(
		recovery.Recovery(),
		observability.Server(
			observability.WithLogRequestBody(true),
			observability.WithLogResponseBody(true),
		),
		auth.Auth(),
	)

	// 启动服务器
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

### 在特定路由组中使用中间件

```go
// 创建路由组
api := server.Group("/api")

// 在路由组上使用中间件
api.UseMiddleware(
	auth.Auth(
		auth.WithValidator(auth.NewJWTValidator("your-secret", "")),
	),
)

// 添加路由
api.GET("/users", func(c *mist.Context) {
	c.RespondWithJSON(200, map[string]string{"users": "list"})
})
```

## 创建自定义中间件

你可以创建自定义中间件来满足特定需求：

```go
package main

import (
	"context"
	"time"
	
	"github.com/dormoron/phantasm/middleware"
)

// 自定义中间件：超时控制
func TimeoutMiddleware(timeout time.Duration) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 创建带超时的上下文
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			
			// 调用下一个处理程序
			return handler(ctx, req)
		}
	}
}

// 使用自定义中间件
server.UseMiddleware(
	TimeoutMiddleware(time.Second * 5), // 5秒超时
)
```

## 链式中间件

你可以使用`middleware.Chain`函数将多个中间件组合在一起：

```go
// 链式中间件
chain := middleware.Chain(
	recovery.Recovery(),
	logging.Logging(),
	auth.Auth(),
)

// 应用到处理程序
handler := chain(yourHandler)
```

## 中间件最佳实践

1. **中间件顺序很重要**：
   - 将恢复(recovery)中间件放在最外层，以捕获其他中间件中的panic
   - 将日志/跟踪中间件放在较外层，以记录所有请求
   - 将认证/授权中间件放在核心业务逻辑之前

2. **性能考虑**：
   - 对于高性能场景，选择性地记录请求和响应体
   - 使用`WithSkipper`跳过对静态资源等不需要详细日志的请求
   - 限制记录的消息体大小，避免日志过大

3. **错误处理**：
   - 确保中间件正确传播错误
   - 使用recovery中间件防止服务崩溃
   - 为不同的错误场景提供清晰的错误信息

4. **可观测性**：
   - 在生产环境中，考虑使用`observability`中间件以获得完整的可观测能力
   - 为每个请求添加唯一标识符，以便在日志、指标和跟踪中关联信息
   - 设置适当的日志级别，避免过多的调试日志在生产环境中产生 

## 限流中间件 (Rate Limiter)

限流中间件提供了多种算法来控制请求速率，防止过度请求对服务造成压力。

### 特点

- 支持多种限流算法：令牌桶、滑动窗口和GCRA
- 可自定义限流标识键（默认使用客户端IP）
- 提供可配置的失败处理
- 自动清理不活跃的限流数据
- 详细的日志记录

### 使用方法

```go
import (
    "phantasm/middleware/limiter"
)

// 1. 使用默认配置 - 使用令牌桶算法(100 QPS)
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit()),
)

// 2. 使用令牌桶限流器 - 每秒50个请求，最大突发100
tokenBucketLimiter := limiter.NewTokenBucketLimiter(50, 100)
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit(
        limiter.WithLimiter(tokenBucketLimiter),
    )),
)

// 3. 使用滑动窗口限流器 - 10秒内允许100个请求
slidingWindowLimiter := limiter.NewSlidingWindowLimiter(100, 10*time.Second)
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit(
        limiter.WithLimiter(slidingWindowLimiter),
    )),
)

// 4. 使用GCRA限流器 - 每秒20个请求，允许5个请求的突发
gcraLimiter := limiter.NewGCRALimiter(20, 5)
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit(
        limiter.WithLimiter(gcraLimiter),
    )),
)

// 5. 自定义限流键生成函数 - 基于用户ID限流
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit(
        limiter.WithKeyFunc(func(ctx context.Context) string {
            if userID, ok := ctx.Value("user_id").(string); ok {
                return userID
            }
            return "anonymous"
        }),
    )),
)

// 6. 自定义限流失败处理
srv := server.NewServer(
    server.WithMiddleware(limiter.Limit(
        limiter.WithFailureHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
            // 返回自定义响应
            return &api.LimitExceededResponse{
                Message: "请稍后再试",
                RetryAfter: 30,
            }, nil
        }),
    )),
)
```

### 配置选项

- `WithLimiter(limiter)` - 设置自定义限流器
- `WithLogger(logger)` - 设置日志记录器
- `WithKeyFunc(keyFunc)` - 设置限流键生成函数
- `WithFailureHandler(handler)` - 设置限流失败处理函数

### 限流算法说明

1. **令牌桶算法**：以恒定速率向桶中添加令牌，每个请求消耗一个令牌，适合处理突发流量
2. **滑动窗口算法**：在指定时间窗口内限制请求数量，提供更精确的流量控制
3. **GCRA算法**：通用单元速率算法，适合精确控制请求间隔，确保请求平滑分布 