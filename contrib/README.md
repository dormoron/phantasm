# 扩展组件

本目录包含Phantasm框架的各种扩展组件，用于集成外部系统和提供额外功能。

## 目录结构

- `config/`: 配置中心实现
  - `nacos/`: 基于Nacos的配置中心实现
- `encoding/`: 编码器实现
  - `proto/`: Protocol Buffers编码器
- `metrics/`: 指标收集实现
  - `prometheus/`: 基于Prometheus的指标收集
- `middleware/`: 中间件实现
  - `ratelimit/`: 限流中间件
- `registry/`: 服务注册与发现
  - `consul/`: 基于Consul的服务注册与发现
  - `etcd/`: 基于Etcd的服务注册与发现
  - `zookeeper/`: 基于ZooKeeper的服务注册与发现
- `tracing/`: 分布式追踪
  - `jaeger/`: 基于Jaeger的分布式追踪

## 通用设计原则

所有扩展组件都遵循以下设计原则：

1. **统一接口**: 每个组件都实现框架核心定义的接口，确保可替换性
2. **可选配置**: 通过选项模式提供灵活的配置
3. **资源管理**: 所有组件负责正确地管理自己的资源，并提供关闭方法
4. **日志记录**: 支持集成框架的日志系统
5. **无侵入性**: 扩展组件不应对核心组件产生侵入性影响

## 使用方法

### 配置中心

```go
import (
    "phantasm/config"
    "phantasm/contrib/config/nacos"
)

// 创建Nacos配置源
source, err := nacos.NewSource(
    nacos.WithServerAddresses([]string{"127.0.0.1"}),
    nacos.WithDataIDs([]string{"app.json"}),
    nacos.WithNamespaceID("public"),
)
if err != nil {
    // 处理错误
}

// 创建配置管理器
c := config.New(config.WithSource(source))

// 加载配置
if err := c.Load(); err != nil {
    // 处理错误
}

// 获取配置值
var serverConfig struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
if err := c.Scan(&serverConfig); err != nil {
    // 处理错误
}
```

### 指标收集

```go
import (
    "phantasm/contrib/metrics/prometheus"
)

// 创建Prometheus指标收集器
p := prometheus.NewPrometheus(
    prometheus.WithAddress(":9090"),
    prometheus.WithPath("/metrics"),
)

// 创建并注册指标
counter := p.CreateCounter(metrics.CounterOptions{
    Name: "request_total",
    Help: "请求总数",
})

// 增加计数
counter.Inc()

// 启动指标服务器
if err := p.Start(context.Background()); err != nil {
    // 处理错误
}

// 退出时关闭
defer p.Stop(context.Background())
```

### 分布式追踪

```go
import (
    "phantasm/contrib/tracing/jaeger"
)

// 创建Jaeger追踪器
tracer, err := jaeger.NewTracer(
    jaeger.WithServiceName("user-service"),
    jaeger.WithEndpoint("http://localhost:14268/api/traces"),
)
if err != nil {
    // 处理错误
}
defer tracer.Close()

// 创建span
ctx, span := tracer.Start(ctx, "查询用户")
defer span.End()

// 添加属性
span.SetAttributes(attribute.String("user.id", "123"))

// 记录事件
span.AddEvent("开始数据库查询")
```

### 服务注册与发现

```go
import (
    "phantasm/contrib/registry/etcd"
)

// 创建Etcd客户端
client, err := clientv3.New(clientv3.Config{
    Endpoints: []string{"127.0.0.1:2379"},
})
if err != nil {
    // 处理错误
}

// 创建注册中心
reg := etcd.NewRegistry(client,
    etcd.WithPrefix("/services"),
    etcd.WithTTL(time.Second * 10),
)

// 注册服务
err = reg.Register(context.Background(), &registry.ServiceInstance{
    ID:        "instance-001",
    Name:      "user-service",
    Version:   "v1.0.0",
    Endpoints: []string{"http://localhost:8080"},
})
if err != nil {
    // 处理错误
}

// 发现服务
instances, err := reg.GetService(context.Background(), "user-service")
if err != nil {
    // 处理错误
}

// 监听服务变更
watcher, err := reg.Watch(context.Background(), "user-service")
if err != nil {
    // 处理错误
}
defer watcher.Stop()

// 处理服务变更
for {
    services, err := watcher.Next()
    if err != nil {
        break
    }
    // 处理服务列表变更
}
```

## 贡献组件

要贡献新的组件，请遵循以下步骤：

1. 确保组件实现了框架定义的相应接口
2. 通过选项模式提供配置
3. 编写完整的单元测试
4. 提供详细的README文档和示例
5. 遵循项目的代码风格和命名约定 