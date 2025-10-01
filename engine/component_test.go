package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockComponent 用于测试的模拟组件
type MockComponent struct {
	name        string
	executeFunc func(ctx context.Context, data map[string]interface{}) error
}

func (m *MockComponent) Name() string {
	return m.name
}

func (m *MockComponent) Execute(ctx context.Context, data map[string]interface{}) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, data)
	}
	return nil
}

// MockInitializableComponent 用于测试的可初始化组件
type MockInitializableComponent struct {
	MockComponent
	initializeFunc func(ctx context.Context) error
}

func (m *MockInitializableComponent) Initialize(ctx context.Context) error {
	if m.initializeFunc != nil {
		return m.initializeFunc(ctx)
	}
	return nil
}

// MockCleanupComponent 用于测试的可清理组件
type MockCleanupComponent struct {
	MockComponent
	cleanupFunc func(ctx context.Context) error
}

func (m *MockCleanupComponent) Cleanup(ctx context.Context) error {
	if m.cleanupFunc != nil {
		return m.cleanupFunc(ctx)
	}
	return nil
}

// MockRetryableComponent 用于测试的可重试组件
type MockRetryableComponent struct {
	MockComponent
	shouldRetryFunc func(err error) bool
	retryConfig     RetryConfig
}

func (m *MockRetryableComponent) ShouldRetry(err error) bool {
	if m.shouldRetryFunc != nil {
		return m.shouldRetryFunc(err)
	}
	return false
}

func (m *MockRetryableComponent) GetRetryConfig() RetryConfig {
	return m.retryConfig
}

// MockValidatableComponent 用于测试的可验证组件
type MockValidatableComponent struct {
	MockComponent
	validateFunc func() error
}

func (m *MockValidatableComponent) Validate() error {
	if m.validateFunc != nil {
		return m.validateFunc()
	}
	return nil
}

// MockComponentFactory 用于测试的组件工厂
type MockComponentFactory struct {
	componentType string
	createFunc    func(config ComponentConfig) (Component, error)
}

func (f *MockComponentFactory) Create(config ComponentConfig) (Component, error) {
	if f.createFunc != nil {
		return f.createFunc(config)
	}
	return &MockComponent{name: config.Name}, nil
}

func (f *MockComponentFactory) GetType() string {
	return f.componentType
}

func TestRetryConfig(t *testing.T) {
	t.Run("RetryConfig creation", func(t *testing.T) {
		config := RetryConfig{
			MaxRetries: 3,
			Delay:      time.Second,
			Backoff:    2.0,
		}

		if config.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
		}
		if config.Delay != time.Second {
			t.Errorf("Expected Delay to be 1s, got %v", config.Delay)
		}
		if config.Backoff != 2.0 {
			t.Errorf("Expected Backoff to be 2.0, got %f", config.Backoff)
		}
	})
}

func TestComponentConfig(t *testing.T) {
	t.Run("ComponentConfig creation", func(t *testing.T) {
		retryConfig := &RetryConfig{
			MaxRetries: 3,
			Delay:      time.Second,
			Backoff:    2.0,
		}

		config := ComponentConfig{
			Name:         "test-component",
			Type:         "test-type",
			Config:       map[string]interface{}{"key": "value"},
			Dependencies: []string{"dep1", "dep2"},
			Timeout:      30 * time.Second,
			Retry:        retryConfig,
			Critical:     true,
			Enabled:      true,
		}

		if config.Name != "test-component" {
			t.Errorf("Expected Name to be 'test-component', got %s", config.Name)
		}
		if config.Type != "test-type" {
			t.Errorf("Expected Type to be 'test-type', got %s", config.Type)
		}
		if len(config.Dependencies) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(config.Dependencies))
		}
		if !config.Critical {
			t.Error("Expected Critical to be true")
		}
		if !config.Enabled {
			t.Error("Expected Enabled to be true")
		}
	})
}

func TestComponentRegistry(t *testing.T) {
	t.Run("NewComponentRegistry", func(t *testing.T) {
		registry := NewComponentRegistry()
		if registry == nil {
			t.Error("Expected registry to be non-nil")
		}
		if registry.factories == nil {
			t.Error("Expected factories map to be initialized")
		}
	})

	t.Run("Register factory", func(t *testing.T) {
		registry := NewComponentRegistry()
		factory := &MockComponentFactory{componentType: "test-type"}

		registry.Register(factory)

		types := registry.GetRegisteredTypes()
		if len(types) != 1 {
			t.Errorf("Expected 1 registered type, got %d", len(types))
		}
		if types[0] != "test-type" {
			t.Errorf("Expected registered type to be 'test-type', got %s", types[0])
		}
	})

	t.Run("Create component with registered factory", func(t *testing.T) {
		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{name: config.Name}, nil
			},
		}

		registry.Register(factory)

		config := ComponentConfig{
			Name: "test-component",
			Type: "test-type",
		}

		component, err := registry.Create(config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if component == nil {
			t.Error("Expected component to be non-nil")
		}
		if component.Name() != "test-component" {
			t.Errorf("Expected component name to be 'test-component', got %s", component.Name())
		}
	})

	t.Run("Create component with unregistered type", func(t *testing.T) {
		registry := NewComponentRegistry()

		config := ComponentConfig{
			Name: "test-component",
			Type: "unknown-type",
		}

		component, err := registry.Create(config)
		if err == nil {
			t.Error("Expected error for unknown component type")
		}
		if component != nil {
			t.Error("Expected component to be nil for unknown type")
		}
	})

	t.Run("Create component with factory error", func(t *testing.T) {
		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return nil, errors.New("factory error")
			},
		}

		registry.Register(factory)

		config := ComponentConfig{
			Name: "test-component",
			Type: "test-type",
		}

		component, err := registry.Create(config)
		if err == nil {
			t.Error("Expected error from factory")
		}
		if component != nil {
			t.Error("Expected component to be nil when factory returns error")
		}
	})

	t.Run("GetRegisteredTypes with multiple factories", func(t *testing.T) {
		registry := NewComponentRegistry()
		factory1 := &MockComponentFactory{componentType: "type1"}
		factory2 := &MockComponentFactory{componentType: "type2"}

		registry.Register(factory1)
		registry.Register(factory2)

		types := registry.GetRegisteredTypes()
		if len(types) != 2 {
			t.Errorf("Expected 2 registered types, got %d", len(types))
		}

		// Check that both types are present (order may vary)
		typeMap := make(map[string]bool)
		for _, t := range types {
			typeMap[t] = true
		}

		if !typeMap["type1"] {
			t.Error("Expected 'type1' to be registered")
		}
		if !typeMap["type2"] {
			t.Error("Expected 'type2' to be registered")
		}
	})
}

func TestComponentInterfaces(t *testing.T) {
	t.Run("Component interface", func(t *testing.T) {
		component := &MockComponent{name: "test"}
		
		if component.Name() != "test" {
			t.Errorf("Expected name to be 'test', got %s", component.Name())
		}

		err := component.Execute(context.Background(), nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("InitializableComponent interface", func(t *testing.T) {
		component := &MockInitializableComponent{
			MockComponent: MockComponent{name: "test"},
		}

		// Test Component interface methods
		if component.Name() != "test" {
			t.Errorf("Expected name to be 'test', got %s", component.Name())
		}

		// Test InitializableComponent interface method
		err := component.Initialize(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("CleanupComponent interface", func(t *testing.T) {
		component := &MockCleanupComponent{
			MockComponent: MockComponent{name: "test"},
		}

		// Test Component interface methods
		if component.Name() != "test" {
			t.Errorf("Expected name to be 'test', got %s", component.Name())
		}

		// Test CleanupComponent interface method
		err := component.Cleanup(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RetryableComponent interface", func(t *testing.T) {
		retryConfig := RetryConfig{
			MaxRetries: 3,
			Delay:      time.Second,
			Backoff:    2.0,
		}

		component := &MockRetryableComponent{
			MockComponent: MockComponent{name: "test"},
			retryConfig:   retryConfig,
		}

		// Test Component interface methods
		if component.Name() != "test" {
			t.Errorf("Expected name to be 'test', got %s", component.Name())
		}

		// Test RetryableComponent interface methods
		if component.ShouldRetry(errors.New("test error")) {
			t.Error("Expected ShouldRetry to return false by default")
		}

		config := component.GetRetryConfig()
		if config.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
		}
	})

	t.Run("ValidatableComponent interface", func(t *testing.T) {
		component := &MockValidatableComponent{
			MockComponent: MockComponent{name: "test"},
		}

		// Test Component interface methods
		if component.Name() != "test" {
			t.Errorf("Expected name to be 'test', got %s", component.Name())
		}

		// Test ValidatableComponent interface method
		err := component.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}