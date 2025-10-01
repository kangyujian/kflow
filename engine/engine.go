package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExecutionStats 执行统计信息
type ExecutionStats struct {
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	LayersTotal   int                    `json:"layers_total"`
	LayersSuccess int                    `json:"layers_success"`
	LayersFailed  int                    `json:"layers_failed"`
	LayerStats    map[string]*LayerStats `json:"layer_stats"`
	Success       bool                   `json:"success"`
	Error         error                  `json:"error,omitempty"`
}

// LayerStats 层级统计信息
type LayerStats struct {
	Name              string        `json:"name"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	Duration          time.Duration `json:"duration"`
	ComponentsTotal   int           `json:"components_total"`
	ComponentsSuccess int           `json:"components_success"`
	ComponentsFailed  int           `json:"components_failed"`
	Success           bool          `json:"success"`
	Error             error         `json:"error,omitempty"`
}

// Engine DAG 执行引擎
type Engine struct {
	config       *Config
	layers       []*Layer
	registry     *ComponentRegistry
	logger       Logger
	errorHandler ErrorHandler
	middleware   []Middleware
	mu           sync.RWMutex
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// ErrorHandler 错误处理接口
type ErrorHandler interface {
	HandleError(ctx context.Context, err error, component string, layer string) error
}

// Middleware 中间件接口
type Middleware interface {
	BeforeExecution(ctx context.Context, config *Config) error
	AfterExecution(ctx context.Context, stats *ExecutionStats) error
	BeforeLayer(ctx context.Context, layer *Layer) error
	AfterLayer(ctx context.Context, layer *Layer, stats *LayerStats) error
}

// EngineOption 引擎选项
type EngineOption func(*Engine)

// WithLogger 设置日志器
func WithLogger(logger Logger) EngineOption {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithErrorHandler 设置错误处理器
func WithErrorHandler(handler ErrorHandler) EngineOption {
	return func(e *Engine) {
		e.errorHandler = handler
	}
}

// WithMiddleware 添加中间件
func WithMiddleware(middleware ...Middleware) EngineOption {
	return func(e *Engine) {
		e.middleware = append(e.middleware, middleware...)
	}
}

// NewEngine 创建新的执行引擎
func NewEngine(config *Config, registry *ComponentRegistry, options ...EngineOption) (*Engine, error) {
	if config == nil {
		return nil, &ConfigError{
			Type:    "nil_config",
			Message: "config cannot be nil",
		}
	}

	if registry == nil {
		return nil, &ConfigError{
			Type:    "nil_registry",
			Message: "component registry cannot be nil",
		}
	}

	engine := &Engine{
		config:       config,
		registry:     registry,
		layers:       make([]*Layer, 0, len(config.Layers)),
		logger:       &defaultLogger{},
		errorHandler: &defaultErrorHandler{},
	}

	// 应用选项
	for _, option := range options {
		option(engine)
	}

	// 创建层级实例
	for _, layerConfig := range config.Layers {
		layer, err := NewLayer(layerConfig, registry)
		if err != nil {
			return nil, fmt.Errorf("failed to create layer %s: %w", layerConfig.Name, err)
		}

		// 验证层级
		if err := layer.Validate(); err != nil {
			return nil, fmt.Errorf("layer validation failed for %s: %w", layerConfig.Name, err)
		}

		engine.layers = append(engine.layers, layer)
	}

	return engine, nil
}

// Execute 执行 DAG
func (e *Engine) Execute(ctx context.Context, data map[string]interface{}) (*ExecutionStats, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	stats := &ExecutionStats{
		StartTime:   time.Now(),
		LayersTotal: len(e.layers),
		LayerStats:  make(map[string]*LayerStats),
	}

	e.logger.Info("Starting DAG execution", "dag", e.config.Name, "layers", len(e.layers))

	// 执行前置中间件
	for _, middleware := range e.middleware {
		if err := middleware.BeforeExecution(ctx, e.config); err != nil {
			return stats, fmt.Errorf("middleware before execution failed: %w", err)
		}
	}

	// 设置全局超时
	if e.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	// 执行层级
	var executionError error
	for _, layer := range e.layers {
		layerStats := &LayerStats{
			Name:            layer.Name(),
			StartTime:       time.Now(),
			ComponentsTotal: len(layer.Components()),
		}
		stats.LayerStats[layer.Name()] = layerStats

		e.logger.Info("Executing layer", "layer", layer.Name(), "mode", layer.Mode())

		// 执行层级前置中间件
		for _, middleware := range e.middleware {
			if err := middleware.BeforeLayer(ctx, layer); err != nil {
				e.logger.Error("Middleware before layer failed", "layer", layer.Name(), "error", err)
				layerStats.Error = err
				executionError = err
				break
			}
		}

		if executionError == nil {
			// 执行层级
			if err := layer.Execute(ctx, data); err != nil {
				e.logger.Error("Layer execution failed", "layer", layer.Name(), "error", err)
				layerStats.Error = err
				layerStats.Success = false
				stats.LayersFailed++
				executionError = err

				// 使用错误处理器处理错误
				if handledErr := e.errorHandler.HandleError(ctx, err, "", layer.Name()); handledErr != nil {
					e.logger.Error("Error handler failed", "layer", layer.Name(), "error", handledErr)
					executionError = handledErr
				}

				// 如果是关键组件错误，停止执行
				if _, ok := err.(*CriticalComponentError); ok {
					e.logger.Error("Critical component failed, stopping execution", "layer", layer.Name())
					break
				}
			} else {
				layerStats.Success = true
				stats.LayersSuccess++
				e.logger.Info("Layer executed successfully", "layer", layer.Name())
			}
		}

		// 更新层级统计
		layerStats.EndTime = time.Now()
		layerStats.Duration = layerStats.EndTime.Sub(layerStats.StartTime)
		layerStats.ComponentsSuccess = layerStats.ComponentsTotal - layerStats.ComponentsFailed

		// 执行层级后置中间件
		for _, middleware := range e.middleware {
			if err := middleware.AfterLayer(ctx, layer, layerStats); err != nil {
				e.logger.Error("Middleware after layer failed", "layer", layer.Name(), "error", err)
			}
		}

		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			executionError = ctx.Err()
			e.logger.Warn("Execution cancelled", "error", executionError)
			break
		default:
		}
	}

	// 更新执行统计
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.Success = executionError == nil
	stats.Error = executionError

	// 执行后置中间件
	for _, middleware := range e.middleware {
		if err := middleware.AfterExecution(ctx, stats); err != nil {
			e.logger.Error("Middleware after execution failed", "error", err)
		}
	}

	if stats.Success {
		e.logger.Info("DAG execution completed successfully",
			"dag", e.config.Name,
			"duration", stats.Duration,
			"layers_success", stats.LayersSuccess)
	} else {
		e.logger.Error("DAG execution failed",
			"dag", e.config.Name,
			"duration", stats.Duration,
			"layers_success", stats.LayersSuccess,
			"layers_failed", stats.LayersFailed,
			"error", executionError)
	}

	return stats, executionError
}

// GetConfig 获取配置
func (e *Engine) GetConfig() *Config {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// GetLayers 获取所有层级
func (e *Engine) GetLayers() []*Layer {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.layers
}

// GetLayer 根据名称获取层级
func (e *Engine) GetLayer(name string) (*Layer, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, layer := range e.layers {
		if layer.Name() == name {
			return layer, true
		}
	}
	return nil, false
}

// Validate 验证引擎配置
func (e *Engine) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 验证配置
	parser := NewConfigParser()
	if err := parser.validateConfig(e.config); err != nil {
		return err
	}

	// 验证所有层级
	for _, layer := range e.layers {
		if err := layer.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (l *defaultLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *defaultLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *defaultLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

// defaultErrorHandler 默认错误处理实现
type defaultErrorHandler struct{}

func (h *defaultErrorHandler) HandleError(ctx context.Context, err error, component string, layer string) error {
	// 默认不做任何处理，直接返回原错误
	return nil
}
