package eidola

import (
	"context"
	"log"
	"time"

	"github.com/dormoron/eidola/registry/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"

	phantasm_registry "github.com/dormoron/phantasm/registry"
)

// ExampleUsage 演示如何使用eidola注册中心适配器
func ExampleUsage() {
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("无法连接到etcd: %v", err)
	}
	defer cli.Close()

	// 创建etcd会话选项
	opts := []concurrency.SessionOption{
		concurrency.WithTTL(30), // 30秒TTL
	}

	// 创建eidola的etcd注册中心
	eidolaRegistry, err := etcd.NewRegistry(cli, opts...)
	if err != nil {
		log.Fatalf("创建注册中心失败: %v", err)
	}
	defer eidolaRegistry.Close()

	// 使用适配器
	registrar, err := EidolaRegistryFactory(eidolaRegistry)
	if err != nil {
		log.Fatalf("创建适配器失败: %v", err)
	}

	// 创建服务实例
	service := &phantasm_registry.ServiceInstance{
		ID:        "service-001",
		Name:      "example-service",
		Version:   "v1.0.0",
		Metadata:  map[string]string{"region": "cn-north"},
		Endpoints: []string{"http://localhost:8080", "grpc://localhost:9000"},
		Status:    phantasm_registry.StatusUp,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 注册服务
	ctx := context.Background()
	if err := registrar.Register(ctx, service); err != nil {
		log.Fatalf("注册服务失败: %v", err)
	}
	log.Println("服务注册成功!")

	// 使用eidola的ListServices接口查询服务
	instances, err := eidolaRegistry.ListServices(ctx, service.Name)
	if err != nil {
		log.Fatalf("查询服务失败: %v", err)
	}
	log.Printf("查询到%d个服务实例", len(instances))

	// 注销服务
	if err := registrar.Deregister(ctx, service); err != nil {
		log.Fatalf("注销服务失败: %v", err)
	}
	log.Println("服务注销成功!")
}
