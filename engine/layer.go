package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExecutionMode 执行模式
type ExecutionMode string

const (
	// SerialMode 串行执行模式
	SerialMode ExecutionMode = "serial"
	// ParallelMode 并行执行模式
	ParallelMode ExecutionMode = "parallel"
	// AsyncMode 异步执行模式
	AsyncMode ExecutionMode = "async"
)

// LayerConfig 层级配置
type LayerConfig struct {
    Name         string            `json:"name"`
    Mode         ExecutionMode     `json:"mode"`
    Components   []ComponentConfig `json:"components"`
    Timeout      time.Duration     `json:"timeout"`
    Dependencies []string          `json:"dependencies"`
    Enabled      bool              `json:"enabled"`
    Parallel     int               `json:"parallel,omitempty"` // 并行度限制
    Remove       bool              `json:"remove,omitempty"`
}

// Layer 表示 DAG 中的一个层级
type Layer struct {
	config     LayerConfig
	components []Component
	registry   *ComponentRegistry
}

// NewLayer 创建新的层级
func NewLayer(config LayerConfig, registry *ComponentRegistry) (*Layer, error) {
	layer := &Layer{
		config:     config,
		components: make([]Component, 0, len(config.Components)),
		registry:   registry,
	}

	// 创建组件实例
	for _, componentConfig := range config.Components {
		if !componentConfig.Enabled {
			continue
		}

		component, err := registry.Create(componentConfig)
		if err != nil {
			return nil, &ComponentError{
				Type:    "component_creation_failed",
				Message: fmt.Sprintf("failed to create component %s: %v", componentConfig.Name, err),
				Layer:   config.Name,
			}
		}

		layer.components = append(layer.components, component)
	}

	return layer, nil
}

// Name 返回层级名称
func (l *Layer) Name() string {
	return l.config.Name
}

// Mode 返回执行模式
func (l *Layer) Mode() ExecutionMode {
	return l.config.Mode
}

// Components 返回层级中的所有组件
func (l *Layer) Components() []Component {
	return l.components
}

// Execute 执行层级中的所有组件
func (l *Layer) Execute(ctx context.Context, data DataContext) error {
	if !l.config.Enabled {
		return nil
	}

	// 设置超时
	if l.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.config.Timeout)
		defer cancel()
	}

	switch l.config.Mode {
	case SerialMode:
		return l.executeSerial(ctx, data)
	case ParallelMode:
		return l.executeParallel(ctx, data)
	case AsyncMode:
		return l.executeAsync(ctx, data)
	default:
		return &ConfigError{
			Type:    "invalid_execution_mode",
			Message: fmt.Sprintf("unsupported execution mode: %s", l.config.Mode),
			Field:   "mode",
		}
	}
}

// executeSerial 串行执行组件
func (l *Layer) executeSerial(ctx context.Context, data DataContext) error {
	for _, component := range l.components {
		if err := l.executeComponent(ctx, component, data); err != nil {
			return err
		}
	}
	return nil
}

// executeParallel 并行执行组件
func (l *Layer) executeParallel(ctx context.Context, data DataContext) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(l.components))

	// 限制并行度
	parallel := l.config.Parallel
	if parallel <= 0 {
		parallel = len(l.components)
	}

	semaphore := make(chan struct{}, parallel)

	for _, component := range l.components {
		wg.Add(1)
		go func(comp Component) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := l.executeComponent(ctx, comp, data); err != nil {
				errChan <- err
			}
		}(component)
	}

	// 等待所有组件完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)

		// 如果是关键组件错误，立即返回
		if _, ok := err.(*CriticalComponentError); ok {
			return err
		}
	}

	if len(errors) > 0 {
		return &ExecutionError{
			Type:      "parallel_execution_failed",
			Message:   fmt.Sprintf("parallel execution failed with %d errors", len(errors)),
			Layer:     l.config.Name,
			Timestamp: time.Now(),
			Cause:     errors[0],
		}
	}

	return nil
}

// executeAsync 异步执行组件
func (l *Layer) executeAsync(ctx context.Context, data DataContext) error {
	// 异步执行不等待结果，直接启动所有组件
	for _, component := range l.components {
		go func(comp Component) {
			l.executeComponent(ctx, comp, data)
		}(component)
	}
	return nil
}

// executeComponent 执行单个组件
func (l *Layer) executeComponent(ctx context.Context, component Component, data DataContext) error {
	componentName := component.Name()

	// 初始化组件
	if initComp, ok := component.(InitializableComponent); ok {
		if err := initComp.Initialize(ctx); err != nil {
			return &ComponentError{
				Type:      "initialization_failed",
				Message:   fmt.Sprintf("component initialization failed: %v", err),
				Component: componentName,
				Layer:     l.config.Name,
				Cause:     err,
			}
		}
	}

	// 清理组件
	if cleanupComp, ok := component.(CleanupComponent); ok {
		defer func() {
			if err := cleanupComp.Cleanup(ctx); err != nil {
				// 记录清理错误，但不影响主流程
				fmt.Printf("Warning: cleanup failed for component %s: %v\n", componentName, err)
			}
		}()
	}

	// 执行组件
	var err error
	if retryComp, ok := component.(RetryableComponent); ok {
		err = l.executeWithRetry(ctx, retryComp, data)
	} else {
		err = component.Execute(ctx, data)
	}

	if err != nil {
		// 检查是否为关键组件
		if l.isCriticalComponent(componentName) {
			return &CriticalComponentError{
				Component: componentName,
				Layer:     l.config.Name,
				Cause:     err,
			}
		}

		return &ExecutionError{
			Type:      "component_execution_failed",
			Message:   fmt.Sprintf("component execution failed: %v", err),
			Component: componentName,
			Layer:     l.config.Name,
			Timestamp: time.Now(),
			Cause:     err,
		}
	}

	return nil
}

func (l *Layer) executeWithRetry(ctx context.Context, component RetryableComponent, data DataContext) error {
	retryConfig := component.GetRetryConfig()
	var lastErr error
	var retryErrors []error

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// 计算退避延迟
			delay := time.Duration(float64(retryConfig.Delay) * (retryConfig.Backoff * float64(attempt-1)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := component.Execute(ctx, data)
		if err == nil {
			return nil
		}

		lastErr = err
		retryErrors = append(retryErrors, err)

		// 检查是否应该重试
		if !component.ShouldRetry(err) {
			break
		}
	}

	return &RetryExhaustedError{
		Component:   component.Name(),
		MaxRetries:  retryConfig.MaxRetries,
		LastError:   lastErr,
		RetryErrors: retryErrors,
	}
}

// isCriticalComponent 检查组件是否为关键组件
func (l *Layer) isCriticalComponent(componentName string) bool {
	for _, config := range l.config.Components {
		if config.Name == componentName {
			return config.Critical
		}
	}
	return false
}

// Validate 验证层级配置
func (l *Layer) Validate() error {
	if l.config.Name == "" {
		return &ValidationError{
			Field:   "name",
			Value:   l.config.Name,
			Message: "layer name cannot be empty",
		}
	}

	if l.config.Mode == "" {
		return &ValidationError{
			Field:   "mode",
			Value:   l.config.Mode,
			Message: "execution mode cannot be empty",
		}
	}

	validModes := map[ExecutionMode]bool{
		SerialMode:   true,
		ParallelMode: true,
		AsyncMode:    true,
	}

	if !validModes[l.config.Mode] {
		return &ValidationError{
			Field:   "mode",
			Value:   l.config.Mode,
			Message: fmt.Sprintf("invalid execution mode: %s", l.config.Mode),
		}
	}

	// 验证组件
	for _, component := range l.components {
		if validatable, ok := component.(ValidatableComponent); ok {
			if err := validatable.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}
