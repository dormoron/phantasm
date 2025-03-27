package selector

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/dormoron/phantasm/registry"
)

// Node 是一个节点
type Node struct {
	// ID 是节点的唯一标识符
	ID string
	// Address 是节点的地址
	Address string
	// Metadata 是节点的元数据
	Metadata map[string]string
	// Weight 是节点的权重
	Weight int64
}

// FilterFunc 是节点选择过滤器
type FilterFunc func(nodes []Node) []Node

// NodeBuilderFunc 构建节点
type NodeBuilderFunc func(id string, address string, metadata map[string]string) (Node, error)

// Selector 是节点选择器接口
type Selector interface {
	// Select 选择一个节点
	Select(ctx context.Context) (Node, error)
	// Update 更新节点列表
	Update(nodes []Node) error
	// Apply 应用过滤器
	Apply(filters ...FilterFunc)
}

// Option 是选择器选项
type Option func(o *options)

// options 是选择器选项
type options struct {
	filters      []FilterFunc
	nodeBuilder  NodeBuilderFunc
	balancer     BalancerType
	cacheTTL     time.Duration
	subsetSize   int
	healthCheck  bool
	healthParams HealthCheckParams
}

// BalancerType 是负载均衡器接口类型
type BalancerType interface {
	// Pick 从节点列表中选择一个节点
	Pick(ctx context.Context, nodes []Node) (Node, error)
}

var (
	// DefaultNodeBuilder 是默认的节点构建器
	DefaultNodeBuilder = func(id string, address string, metadata map[string]string) (Node, error) {
		return Node{
			ID:       id,
			Address:  address,
			Metadata: metadata,
			Weight:   100,
		}, nil
	}
)

// NewSelector 创建一个新的选择器
func NewSelector(opts ...Option) Selector {
	o := options{
		nodeBuilder: DefaultNodeBuilder,
		balancer:    &Random{},
		cacheTTL:    time.Second * 30,
		subsetSize:  0, // 0表示使用所有节点
		healthCheck: false,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &defaultSelector{
		opts:  o,
		nodes: make([]Node, 0),
	}
}

// defaultSelector 是选择器的默认实现
type defaultSelector struct {
	opts  options
	nodes []Node
}

// Select 选择一个节点
func (s *defaultSelector) Select(ctx context.Context) (Node, error) {
	nodes := s.nodes
	if len(nodes) == 0 {
		return Node{}, ErrNoAvailable
	}

	// 应用过滤器
	for _, f := range s.opts.filters {
		nodes = f(nodes)
		if len(nodes) == 0 {
			return Node{}, ErrNoAvailable
		}
	}

	// 使用均衡器选择节点
	return s.opts.balancer.Pick(ctx, nodes)
}

// Update 更新节点列表
func (s *defaultSelector) Update(nodes []Node) error {
	s.nodes = nodes
	return nil
}

// Apply 应用过滤器
func (s *defaultSelector) Apply(filters ...FilterFunc) {
	s.opts.filters = append(s.opts.filters, filters...)
}

// BuildSelector 从注册中心构建选择器
func BuildSelector(discovery registry.Discovery, serviceName string, opts ...Option) (Selector, error) {
	sel := NewSelector(opts...)
	go func() {
		instances, err := discovery.GetService(context.Background(), serviceName)
		if err != nil {
			return
		}
		nodes, err := instancesToNodes(instances, sel.(*defaultSelector).opts.nodeBuilder)
		if err != nil {
			return
		}
		sel.Update(nodes)
	}()
	return sel, nil
}

// instancesToNodes 转换服务实例为节点
func instancesToNodes(instances []*registry.ServiceInstance, builder NodeBuilderFunc) ([]Node, error) {
	nodes := make([]Node, 0, len(instances))
	for _, ins := range instances {
		for _, endpoint := range ins.Endpoints {
			node, err := builder(ins.ID, endpoint, ins.Metadata)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// WithFilter 选项用于设置过滤器
func WithFilter(filters ...FilterFunc) Option {
	return func(o *options) {
		o.filters = append(o.filters, filters...)
	}
}

// WithNodeBuilder 选项用于设置节点构建器
func WithNodeBuilder(builder NodeBuilderFunc) Option {
	return func(o *options) {
		o.nodeBuilder = builder
	}
}

// WithBalancer 选项用于设置均衡器
func WithBalancer(balancer BalancerType) Option {
	return func(o *options) {
		o.balancer = balancer
	}
}

// WithCacheTTL 选项用于设置缓存TTL
func WithCacheTTL(ttl time.Duration) Option {
	return func(o *options) {
		o.cacheTTL = ttl
	}
}

// WithSubsetSize 选项用于设置子集大小
func WithSubsetSize(size int) Option {
	return func(o *options) {
		o.subsetSize = size
	}
}

// WithHealthCheck 选项用于设置健康检查
func WithHealthCheck(enable bool, params HealthCheckParams) Option {
	return func(o *options) {
		o.healthCheck = enable
		o.healthParams = params
	}
}

// HealthCheckParams 是健康检查参数
type HealthCheckParams struct {
	Interval      time.Duration
	Timeout       time.Duration
	MaxContinuous int
}

// Random 是随机负载均衡器
type Random struct{}

// Pick 随机选择一个节点
func (r *Random) Pick(_ context.Context, nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return Node{}, ErrNoAvailable
	}
	return nodes[nextRandom(int64(len(nodes)))], nil
}

// RoundRobin 是轮询负载均衡器
type RoundRobin struct {
	counter atomic.Int64
}

// Pick 轮询选择一个节点
func (r *RoundRobin) Pick(_ context.Context, nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return Node{}, ErrNoAvailable
	}
	count := r.counter.Add(1)
	return nodes[count%int64(len(nodes))], nil
}

// WeightedRandom 是加权随机负载均衡器
type WeightedRandom struct{}

// Pick 加权随机选择一个节点
func (w *WeightedRandom) Pick(_ context.Context, nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return Node{}, ErrNoAvailable
	}

	// 计算总权重
	var totalWeight int64
	for _, node := range nodes {
		if node.Weight > 0 {
			totalWeight += node.Weight
		}
	}

	if totalWeight <= 0 {
		return nodes[nextRandom(int64(len(nodes)))], nil
	}

	// 加权随机选择
	offset := nextRandom(totalWeight)
	for _, node := range nodes {
		if node.Weight > 0 {
			offset -= node.Weight
			if offset < 0 {
				return node, nil
			}
		}
	}

	// 不应该到达这里
	return nodes[0], nil
}
