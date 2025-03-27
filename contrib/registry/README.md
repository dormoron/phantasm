# 服务注册与发现

Phantasm提供了多种服务注册与发现中心的实现，用于微服务架构中的服务注册和发现。

## 特性

- 支持多种服务注册中心：Memory（内存）、Etcd、Consul、ZooKeeper
- 统一的服务注册接口
- 服务健康检查
- 服务状态自动同步
- 服务实例监控

## 支持的注册中心

### Memory 内存注册中心

内存注册中心适用于开发和测试环境，所有数据存储在内存中，不需要外部依赖。

```go
import (
    "phantasm/contrib/registry/memory"
    "phantasm/registry"
)

// 创建内存注册中心
reg := memory.NewRegistry()

// 使用注册中心
app := phantasm.New(
    phantasm.WithRegistry(reg),
)
```

### Etcd 注册中心

基于Etcd的服务注册与发现中心，提供高可用性和强一致性。

```go
import (
    "phantasm/contrib/registry/etcd"
    "phantasm/registry"
    clientv3 "go.etcd.io/etcd/client/v3"
)

// 创建Etcd客户端
client, err := clientv3.New(clientv3.Config{
    Endpoints: []string{"127.0.0.1:2379"},
})
if err != nil {
    // 处理错误
}

// 创建Etcd注册中心
reg := etcd.NewRegistry(client,
    etcd.WithPrefix("/services"),
    etcd.WithTTL(time.Second * 10),
)

// 使用注册中心
app := phantasm.New(
    phantasm.WithRegistry(reg),
)
```

### Consul 注册中心

基于Consul的服务注册与发现中心，提供服务健康检查和DNS服务发现。

```go
import (
    "phantasm/contrib/registry/consul"
    "phantasm/registry"
    "github.com/hashicorp/consul/api"
)

// 创建Consul客户端
client, err := api.NewClient(&api.Config{
    Address: "127.0.0.1:8500",
})
if err != nil {
    // 处理错误
}

// 创建Consul注册中心
reg := consul.NewRegistry(client,
    consul.WithHealthCheck(true),
    consul.WithTTL(time.Second * 15),
    consul.WithDeregisterCriticalServiceAfter(time.Minute),
)

// 使用注册中心
app := phantasm.New(
    phantasm.WithRegistry(reg),
)
```

### ZooKeeper 注册中心

基于ZooKeeper的服务注册与发现中心，提供可靠的协调服务。

```go
import (
    "phantasm/contrib/registry/zookeeper"
    "phantasm/registry"
)

// 创建ZooKeeper注册中心
reg, err := zookeeper.NewRegistry(
    []string{"127.0.0.1:2181"},
    zookeeper.WithPrefix("/services"),
    zookeeper.WithSessionTimeout(time.Second * 10),
)
if err != nil {
    // 处理错误
}

// 使用注册中心
app := phantasm.New(
    phantasm.WithRegistry(reg),
)
```

## 服务注册

服务实例在应用启动时自动注册，默认情况下，Phantasm会根据应用配置自动构建服务实例：

```go
// 创建应用实例时配置服务信息
app := phantasm.New(
    phantasm.WithName("user-service"),
    phantasm.WithVersion("v1.0.0"),
    phantasm.WithRegistry(reg),
)
```

## 手动注册服务

如果需要手动注册服务，可以使用注册中心的Register方法：

```go
serviceInstance := &registry.ServiceInstance{
    ID:        "instance-001",
    Name:      "user-service",
    Version:   "v1.0.0",
    Endpoints: []string{"http://127.0.0.1:8000", "grpc://127.0.0.1:9000"},
    Metadata: map[string]string{
        "region": "cn-east-1",
        "zone":   "cn-east-1a",
    },
}

// 注册服务
err := reg.Register(context.Background(), serviceInstance)
```

## 服务发现

发现指定服务的所有实例：

```go
// 获取服务实例列表
instances, err := reg.GetService(context.Background(), "user-service")
if err != nil {
    // 处理错误
}

// 使用服务实例
for _, instance := range instances {
    fmt.Printf("ID: %s, Endpoints: %v\n", instance.ID, instance.Endpoints)
}
```

## 监视服务变更

可以通过Watcher接口监视服务实例的变更：

```go
// 创建服务监视器
watcher, err := reg.Watch(context.Background(), "user-service")
if err != nil {
    // 处理错误
}
defer watcher.Stop()

// 持续监视服务变更
for {
    instances, err := watcher.Next()
    if err != nil {
        break
    }
    
    // 处理服务实例变更
    fmt.Printf("服务实例数量: %d\n", len(instances))
}
```

## 优雅关闭

在应用退出时，应停止注册中心以确保资源释放：

```go
// 对于支持Stop方法的注册中心，在应用退出时调用
if reg, ok := reg.(interface{ Stop() error }); ok {
    reg.Stop()
}
```

或者使用Phantasm的内置关闭机制，它会自动处理注册中心的关闭：

```go
// 启动应用
if err := app.Run(); err != nil {
    log.Fatal(err)
}
``` 