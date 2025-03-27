# 内部组件

本目录包含Cosmos框架的内部组件实现，这些组件通常不直接暴露给框架使用者，而是被框架内部使用。

## 目录结构

- `endpoint`: URL端点管理，用于处理和解析服务端点
- `server`: 服务器管理，实现了多服务器的启动和关闭
- `service`: 服务抽象，定义了服务的基本接口和管理器
- `data`: 数据访问层，提供数据库连接和仓库的抽象

## 组件说明

### endpoint

endpoint包提供了处理服务端点URL的工具，包括创建、解析和标准化URL等功能。这个组件主要被registry和transport等模块使用，用于处理服务地址。

```go
// 创建新的端点URL
endpoint := endpoint.NewEndpoint("http", "localhost:8080")

// 从端点列表中解析指定scheme的端点
host, err := endpoint.ParseEndpoint([]string{"http://localhost:8080", "grpc://localhost:9000"}, "grpc")

// 标准化端点URL
normalized, err := endpoint.NormalizeEndpoint("localhost:8080", "http")
```

### server

server包实现了服务器的管理，主要包括Server接口定义和管理多个服务器的Manager。这个组件被app.go使用，用于管理不同类型的服务器。

```go
// 创建服务器管理器
serverManager := server.NewManager(logger)

// 添加HTTP和gRPC服务器
serverManager.Add(httpServer)
serverManager.Add(grpcServer)

// 启动所有服务器
if err := serverManager.Start(ctx); err != nil {
    // 处理错误
}

// 监听系统信号
server.WaitForSignal(logger, func() {
    // 优雅关闭服务
    serverManager.Stop(context.Background())
})
```

### service

service包定义了服务的抽象接口和基础实现，提供了服务生命周期管理和服务工厂模式。这个组件被应用程序用于管理业务服务。

```go
// 创建抽象服务
svc := service.NewAbstractService("user-service", logger)

// 创建服务管理器
serviceManager := service.NewManager(logger)

// 注册服务
serviceManager.Register(userService)
serviceManager.Register(orderService)

// 启动所有服务
serviceManager.Start(ctx)
```

### data

data包提供了数据访问层的抽象，包括数据库连接、事务和仓库模式的实现。这个组件被业务服务使用，用于访问数据库。

```go
// 创建数据库管理器
dbManager := data.NewManager(logger)

// 注册数据库
dbManager.Register(mysqlDB)
dbManager.Register(redisDB)

// 连接所有数据库
if err := dbManager.Connect(ctx); err != nil {
    // 处理错误
}

// 创建仓库
userRepo := data.NewBaseRepository(mysqlDB, "users", logger)
```

## 扩展方式

内部组件设计为可扩展的，可以通过以下方式进行扩展：

1. 实现相应的接口
2. 基于提供的基础实现进行扩展
3. 通过组合使用不同的组件实现新功能

例如，要创建自定义服务器，只需实现`server.Server`接口；要创建自定义数据库实现，只需实现`data.DB`接口。 