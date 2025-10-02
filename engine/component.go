package engine

import (
	"context"
	"time"
)

// Component 定义了 DAG 中组件的基本接口
type Component interface {
	// Name 返回组件的唯一名称
	Name() string

	// Execute 执行组件的主要逻辑
	Execute(ctx context.Context, data DataContext) error
}

// InitializableComponent 可初始化的组件接口
type InitializableComponent interface {
	Component

	// Initialize 在组件执行前进行初始化
	Initialize(ctx context.Context) error
}

// CleanupComponent 可清理的组件接口
type CleanupComponent interface {
	Component

	// Cleanup 在组件执行后进行清理
	Cleanup(ctx context.Context) error
}

// RetryableComponent 可重试的组件接口
type RetryableComponent interface {
	Component

	// ShouldRetry 判断是否应该重试
	ShouldRetry(err error) bool

	// GetRetryConfig 获取重试配置
	GetRetryConfig() RetryConfig
}

// ValidatableComponent 可验证的组件接口
type ValidatableComponent interface {
	Component

	// Validate 验证组件配置
	Validate() error
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int           `json:"max_retries"`
	Delay      time.Duration `json:"delay"`
	Backoff    float64       `json:"backoff"`
}

// ComponentConfig 组件配置
type ComponentConfig struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies"`
	Timeout      time.Duration          `json:"timeout"`
	Retry        *RetryConfig           `json:"retry,omitempty"`
	Critical     bool                   `json:"critical"`
	Enabled      bool                   `json:"enabled"`
}

// ComponentFactory 组件工厂接口
type ComponentFactory interface {
	// Create 根据配置创建组件实例
	Create(config ComponentConfig) (Component, error)

	// GetType 返回工厂支持的组件类型
	GetType() string
}

// ComponentRegistry 组件注册表
type ComponentRegistry struct {
	factories map[string]ComponentFactory
}

// NewComponentRegistry 创建新的组件注册表
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		factories: make(map[string]ComponentFactory),
	}
}

// Register 注册组件工厂
func (r *ComponentRegistry) Register(factory ComponentFactory) {
	r.factories[factory.GetType()] = factory
}

// Create 根据配置创建组件
func (r *ComponentRegistry) Create(config ComponentConfig) (Component, error) {
	factory, exists := r.factories[config.Type]
	if !exists {
		return nil, &ComponentError{
			Type:    "factory_not_found",
			Message: "component factory not found for type: " + config.Type,
		}
	}

	return factory.Create(config)
}

// GetRegisteredTypes 获取所有已注册的组件类型
func (r *ComponentRegistry) GetRegisteredTypes() []string {
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
