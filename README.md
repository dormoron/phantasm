# Phantasm: 优雅、强大的Go微服务框架

Phantasm是一个优雅而强大的Go语言微服务框架，专为构建高性能、高可靠性的微服务应用而设计。框架集成了`mist`作为Web框架和`eidola`作为gRPC框架，提供了从开发到部署的完整解决方案。

[English](./README_EN.md) | 中文

## 🌟 核心特性

- **简洁API设计**：直观易用的API接口，降低学习成本，提高开发效率
- **模块化架构**：松耦合的组件设计，支持灵活组装和定制化开发
- **多协议支持**：同时支持HTTP和gRPC，满足不同场景的通信需求
- **服务治理**：内置服务注册与发现、负载均衡、健康检查等能力
- **全链路追踪**：集成OpenTelemetry，支持分布式追踪、指标收集
- **结构化日志**：基于zap的高性能日志系统，支持多级别、多输出
- **丰富中间件**：提供认证、授权、限流、熔断等常用中间件
- **配置管理**：灵活的配置加载机制，支持多种配置源和动态更新
- **统一错误处理**：标准化的错误定义和处理机制，提高API一致性
- **Buf集成**：内置Buf工具支持，提升Proto文件管理和验证能力
- **优雅启停**：支持平滑启动和关闭，确保服务可靠性
- **开发工具链**：提供命令行工具，简化项目创建和代码生成

## 📦 安装

### 前置条件

- Go 1.23 或更高版本
- 如需使用protobuf功能，需安装protoc
- 可选安装buf工具来增强Proto文件管理体验

### 安装CLI工具

```bash
go install github.com/dormoron/phantasm/cmd/phantasm@latest
```

### 添加框架依赖

```bash
go get github.com/dormoron/phantasm@latest
```

## 🚀 快速开始

### 创建新项目

使用CLI工具快速创建项目：

```bash
# 创建名为"helloworld"的新项目
phantasm new helloworld

# 进入项目目录
cd helloworld
```

### 项目结构

生成的项目结构如下：

```
.
├── api                # 接口定义目录（protobuf文件）
├── cmd                # 命令行入口
│   └── server         # 服务启动入口
├── configs            # 配置文件目录
├── internal           # 内部代码包
│   ├── biz            # 业务逻辑层
│   ├── data           # 数据访问层
│   ├── server         # 服务实现层
│   └── service        # 服务接口层
├── third_party        # 第三方依赖
│   ├── errors         # 标准错误定义
│   ├── buf            # Buf相关配置
│   └── google         # Google API定义
├── buf.yaml           # Buf工具配置
├── go.mod             # Go模块定义
├── go.sum             # Go依赖锁定文件
└── README.md          # 项目说明文档
```

### 启动服务

```bash
# 直接运行
go run ./cmd/server

# 或使用Phantasm工具运行
phantasm run
```

## 💡 核心概念与用法

### 应用生命周期

Phantasm提供完整的应用程序生命周期管理，包括启动前准备、启动、运行中和关闭阶段：

```go
app := phantasm.New(
    phantasm.Name("my-service"),        // 服务名称
    phantasm.Version("v1.0.0"),         // 服务版本
    phantasm.Server(httpServer, grpcServer), // 注册服务器
    phantasm.BeforeStart(func(ctx context.Context) error {
        // 启动前准备工作
        return nil
    }),
    phantasm.AfterStart(func(ctx context.Context) error {
        // 启动成功后的操作
        return nil
    }),
    phantasm.BeforeStop(func(ctx context.Context) error {
        // 停止前的清理工作
        return nil
    }),
    phantasm.AfterStop(func(ctx context.Context) error {
        // 停止后的操作
        return nil
    }),
)

// 启动应用
if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### HTTP服务

基于`mist`框架实现的HTTP服务支持：

```go
import (
    "phantasm/transport/http"
    "github.com/dormoron/mist"
)

// 创建HTTP服务器
httpServer, err := http.NewHTTPServer(
    http.WithAddress(":8000"),
    http.WithTimeout(time.Second*5),
    http.WithMiddleware(
        middleware.Recovery(),
        middleware.Logger(),
    ),
)
if err != nil {
    log.Fatal(err)
}

// 注册路由
httpServer.GET("/hello", func(c *mist.Context) {
    c.RespondWithJSON(200, map[string]interface{}{
        "message": "Hello, Phantasm!",
    })
})

// 注册服务并启动
app := phantasm.New(
    phantasm.Name("http-service"),
    phantasm.Server(httpServer),
)

if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### gRPC服务

基于`eidola`框架实现的gRPC服务支持：

```go
import (
    "phantasm/transport/grpc"
    helloworldpb "myapp/api/helloworld/v1"
)

// 创建gRPC服务器
grpcServer, err := grpc.NewGRPCServer(
    grpc.WithAddress(":9000"),
    grpc.WithTimeout(time.Second*5),
    grpc.WithMiddleware(
        grpcmiddleware.Recovery(),
        grpcmiddleware.Logger(),
    ),
)
if err != nil {
    log.Fatal(err)
}

// 注册服务
helloworldpb.RegisterGreeterServer(grpcServer.Server(), &GreeterService{})

// 注册服务并启动
app := phantasm.New(
    phantasm.Name("grpc-service"),
    phantasm.Server(grpcServer),
)

if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### 服务注册与发现

支持多种注册中心，包括Consul、Etcd、Zookeeper等：

```go
import (
    "phantasm/registry"
    "phantasm/registry/consul"
)

// 创建Consul注册中心
r, err := consul.New(consul.WithAddress("127.0.0.1:8500"))
if err != nil {
    log.Fatal(err)
}

// 在应用程序中使用
app := phantasm.New(
    phantasm.Name("my-service"),
    phantasm.Version("v1.0.0"),
    phantasm.Server(httpServer, grpcServer),
    phantasm.Registrar(r),
)
```

### 配置管理

基于灵活的配置管理系统：

```go
import (
    "phantasm/config"
    "phantasm/config/file"
)

// 加载配置文件
c, err := config.Load(
    file.NewSource("configs/config.yaml"),
    file.NewSource("configs/config.local.yaml", file.WithOptional(true)),
)
if err != nil {
    log.Fatal(err)
}

// 获取配置值
httpPort := c.Get("server.http.port").String("8000")
```

### 日志系统

高性能的结构化日志系统：

```go
import (
    "phantasm/log"
)

// 创建日志记录器
logger := log.NewLogger()

// 记录不同级别的日志
logger.Debug("调试信息", log.String("key", "value"))
logger.Info("普通信息", log.Int("count", 100))
logger.Warn("警告信息", log.Duration("elapsed", time.Second))
logger.Error("错误信息", log.Err(err))
```

### 错误处理

Phantasm提供了标准化的错误处理机制，使API错误响应保持一致：

```go
import (
    "github.com/dormoron/phantasm/errors"
)

// 创建新错误
err := errors.New(
    errors.CodeNotFound,    // 错误代码
    "user.not_found",       // 错误原因
    "用户不存在",            // 错误消息
)

// 添加元数据
err = err.WithMetadata(map[string]string{
    "user_id": "123456",
})

// 在HTTP/gRPC响应中使用
if user == nil {
    return nil, err
}

// 从错误中提取信息
if e, ok := errors.FromError(err); ok {
    code := e.Code      // 获取错误代码
    reason := e.Reason  // 获取错误原因
    message := e.Message // 获取错误消息
}
```

### Buf 工具集成

Phantasm内置了对Buf工具的支持，提供了更好的Proto文件管理体验：

```bash
# 初始化Buf工作区
phantasm proto init

# 使用buf验证Proto文件
phantasm proto lint

# 使用buf检查破坏性变更
phantasm proto breaking

# 使用buf生成代码
phantasm proto generate
```

也可以直接使用项目中提供的配置运行原生buf命令：

```bash
buf lint
buf build
buf generate
```

## 🔧 更多示例

### 使用资源定义

利用Google API提供的资源定义增强API表达能力：

```protobuf
syntax = "proto3";

package api.example.v1;

import "google/api/resource.proto";
import "google/api/field_behavior.proto";

message User {
  option (google.api.resource) = {
    type: "example.com/User"
    pattern: "users/{user_id}"
    name_field: "name"
  };

  // 资源名称
  string name = 1 [(google.api.field_behavior) = IDENTIFIER];
  
  // 用户名
  string username = 2 [(google.api.field_behavior) = REQUIRED];
}
```

### 使用标准错误

在API中使用标准错误响应：

```protobuf
syntax = "proto3";

package api.example.v1;

import "third_party/errors/errors.proto";

message GetUserResponse {
  // 用户数据
  User user = 1;
  
  // 标准错误响应
  errors.ErrorResponse error = 2;
}
```

## 🔧 工具链

### 项目创建

```bash
# 创建标准项目
phantasm new myproject

# 创建纯HTTP项目
phantasm new myproject --http-only

# 创建纯gRPC项目
phantasm new myproject --grpc-only
```

### Protobuf代码生成

```bash
# 生成protobuf代码
phantasm proto generate

# 生成HTTP网关
phantasm proto generate --with-http-gateway
```

### 运行项目

```bash
# 运行项目
phantasm run

# 在指定环境下运行
phantasm run --env=production
```

### 升级框架

```bash
# 升级框架到最新版本
phantasm upgrade
```

## 📚 项目结构

```
├── api/                 # API定义（Protobuf）
├── cmd/                 # 命令行工具
│   ├── phantasm/          # CLI工具
│   └── protoc-gen-phantasm-http/ # Protobuf代码生成器
├── config/              # 配置管理
├── contrib/             # 第三方集成
├── encoding/            # 编码器
├── errors/              # 错误处理
├── examples/            # 示例代码
├── internal/            # 内部包
├── log/                 # 日志
├── metadata/            # 元数据
├── middleware/          # 中间件
├── registry/            # 注册中心
├── selector/            # 负载均衡
├── tools/               # 开发工具
├── transport/           # 传输层
│   ├── grpc/            # gRPC传输
│   └── http/            # HTTP传输
├── app.go               # 应用程序入口
├── options.go           # 选项定义
└── version.go           # 版本信息
```

## 📊 性能基准

在标准测试环境下（8核CPU，16GB内存），Phantasm框架的性能表现：

- HTTP服务: ~50,000 QPS
- gRPC服务: ~100,000 QPS
- 内存占用: ~20MB（基础启动）
- 启动时间: <500ms

## 🔍 与其他框架对比

| 特性 | Phantasm | Go-Kit | Go-Micro | Kratos |
|------|--------|--------|----------|--------|
| 易用性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| 功能完整性 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 社区支持 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 文档质量 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

## 🤝 贡献指南

我们非常欢迎社区贡献，无论是报告问题、提出建议还是提交代码，都可以帮助Phantasm变得更好！

详细的贡献指南请参阅[CONTRIBUTING.md](CONTRIBUTING.md)。

## 📄 许可证

Phantasm框架基于[MIT许可证](LICENSE)开源。

## 🔗 相关项目

- [mist](https://github.com/dormoron/mist) - 轻量级HTTP框架
- [eidola](https://github.com/dormoron/eidola) - 强大的gRPC框架
- [phantasm-examples](https://github.com/dormoron/Phantasm-examples) - 示例项目集合

## 📮 联系我们

- GitHub Issues: [https://github.com/dormoron/phantasm/issues](https://github.com/dormoron/Phantasm/issues)
- 邮箱: dormoron@163.com.com
- 社区: [https://discord.gg/phantasm-framework](https://discord.gg/Phantasm-framework)