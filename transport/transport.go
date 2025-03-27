package transport

import (
	"context"
	"net/url"
)

// Server 是传输服务器接口
type Server interface {
	// Start 启动服务器
	Start(ctx context.Context) error
	// Stop 停止服务器
	Stop(ctx context.Context) error
}

// Endpointer 是可以返回端点信息的服务器接口
type Endpointer interface {
	// Endpoint 返回服务器的端点
	Endpoint() (*url.URL, error)
}

// Handler 是请求处理程序
type Handler interface{}

// Middleware 是服务中间件函数
type Middleware func(Handler) Handler
