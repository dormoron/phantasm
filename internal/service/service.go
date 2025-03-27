package service

import (
	"context"

	"github.com/dormoron/phantasm/log"
)

// Service 定义了一个服务的基本接口
type Service interface {
	// Init 初始化服务
	Init() error
	// Start 启动服务
	Start(context.Context) error
	// Stop 停止服务
	Stop(context.Context) error
	// Name 获取服务名称
	Name() string
}

// Factory 服务工厂接口
type Factory interface {
	// Create 创建服务实例
	Create() (Service, error)
	// Type 获取服务类型
	Type() string
}

// AbstractService 提供服务的基本实现
type AbstractService struct {
	name   string
	logger log.Logger
}

// NewAbstractService 创建抽象服务
func NewAbstractService(name string, logger log.Logger) *AbstractService {
	return &AbstractService{
		name:   name,
		logger: logger,
	}
}

// Init 初始化服务（默认实现）
func (s *AbstractService) Init() error {
	s.logger.Info("服务初始化", log.String("service", s.name))
	return nil
}

// Start 启动服务（默认实现）
func (s *AbstractService) Start(ctx context.Context) error {
	s.logger.Info("服务启动", log.String("service", s.name))
	return nil
}

// Stop 停止服务（默认实现）
func (s *AbstractService) Stop(ctx context.Context) error {
	s.logger.Info("服务停止", log.String("service", s.name))
	return nil
}

// Name 获取服务名称
func (s *AbstractService) Name() string {
	return s.name
}

// Logger 获取服务日志器
func (s *AbstractService) Logger() log.Logger {
	return s.logger
}

// Manager 服务管理器
type Manager interface {
	// Register 注册服务
	Register(Service) error
	// Get 获取指定服务
	Get(name string) (Service, bool)
	// Start 启动所有服务
	Start(context.Context) error
	// Stop 停止所有服务
	Stop(context.Context) error
}

// manager 服务管理器实现
type manager struct {
	services map[string]Service
	logger   log.Logger
}

// NewManager 创建服务管理器
func NewManager(logger log.Logger) Manager {
	return &manager{
		services: make(map[string]Service),
		logger:   logger,
	}
}

// Register 注册服务
func (m *manager) Register(svc Service) error {
	name := svc.Name()
	m.logger.Debug("注册服务", log.String("service", name))

	if _, exists := m.services[name]; exists {
		m.logger.Warn("服务已存在，将被覆盖", log.String("service", name))
	}

	m.services[name] = svc
	return nil
}

// Get 获取指定服务
func (m *manager) Get(name string) (Service, bool) {
	svc, ok := m.services[name]
	return svc, ok
}

// Start 启动所有服务
func (m *manager) Start(ctx context.Context) error {
	m.logger.Info("启动所有服务", log.Int("count", len(m.services)))

	// 初始化所有服务
	for name, svc := range m.services {
		if err := svc.Init(); err != nil {
			m.logger.Error("服务初始化失败", log.String("service", name), log.String("error", err.Error()))
			return err
		}
	}

	// 启动所有服务
	for name, svc := range m.services {
		if err := svc.Start(ctx); err != nil {
			m.logger.Error("服务启动失败", log.String("service", name), log.String("error", err.Error()))
			return err
		}
	}

	return nil
}

// Stop 停止所有服务
func (m *manager) Stop(ctx context.Context) error {
	m.logger.Info("停止所有服务", log.Int("count", len(m.services)))

	var lastErr error
	// 按照注册的相反顺序停止服务
	for name, svc := range m.services {
		if err := svc.Stop(ctx); err != nil {
			m.logger.Error("服务停止失败", log.String("service", name), log.String("error", err.Error()))
			lastErr = err
		}
	}

	return lastErr
}
