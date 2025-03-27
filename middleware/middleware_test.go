package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dormoron/phantasm/middleware"
)

// TestChain 测试中间件链
func TestChain(t *testing.T) {
	// 用于记录执行顺序的切片
	var order []string

	// 创建测试中间件
	middleware1 := func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			order = append(order, "before_middleware1")
			resp, err := handler(ctx, req)
			order = append(order, "after_middleware1")
			return resp, err
		}
	}

	middleware2 := func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			order = append(order, "before_middleware2")
			resp, err := handler(ctx, req)
			order = append(order, "after_middleware2")
			return resp, err
		}
	}

	middleware3 := func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			order = append(order, "before_middleware3")
			resp, err := handler(ctx, req)
			order = append(order, "after_middleware3")
			return resp, err
		}
	}

	// 创建中间件链
	m := middleware.Chain(middleware1, middleware2, middleware3)

	// 创建处理程序
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		order = append(order, "handler")
		return "response", nil
	}

	// 应用中间件链
	h := m(handler)

	// 调用处理程序
	resp, err := h(context.Background(), "request")
	if err != nil {
		t.Errorf("handler error: %v", err)
	}

	// 验证响应
	if resp != "response" {
		t.Errorf("expected response 'response', got '%v'", resp)
	}

	// 验证执行顺序
	expected := []string{
		"before_middleware1",
		"before_middleware2",
		"before_middleware3",
		"handler",
		"after_middleware3",
		"after_middleware2",
		"after_middleware1",
	}

	if len(order) != len(expected) {
		t.Errorf("expected %d items in order, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected order[%d] = %s, got %s", i, v, order[i])
		}
	}
}

// TestChainWithError 测试错误处理
func TestChainWithError(t *testing.T) {
	// 创建中间件链
	m := middleware.Chain(
		func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req interface{}) (interface{}, error) {
				resp, err := next(ctx, req)
				// 中间件可以处理或传递错误
				return resp, err
			}
		},
	)

	// 创建会返回错误的处理程序
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("test error")
	}

	// 应用中间件链
	h := m(handler)

	// 调用处理程序
	_, err := h(context.Background(), "request")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("expected error 'test error', got '%v'", err)
	}
}
