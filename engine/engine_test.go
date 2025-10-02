package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	t.Run("Create engine with valid config and registry", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{name: config.Name}, nil
			},
		}
		registry.Register(factory)

		engine, err := NewEngine(config, registry)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if engine == nil {
			t.Error("Expected engine to be non-nil")
		}
	})

	t.Run("Create engine with nil config", func(t *testing.T) {
		registry := NewComponentRegistry()
		engine, err := NewEngine(nil, registry)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		if engine != nil {
			t.Error("Expected engine to be nil")
		}
	})

	t.Run("Create engine with nil registry", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
		}
		engine, err := NewEngine(config, nil)
		if err == nil {
			t.Error("Expected error for nil registry")
		}
		if engine != nil {
			t.Error("Expected engine to be nil")
		}
	})

	t.Run("Create engine with invalid layer", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
			Layers: []LayerConfig{
				{
					Name:    "", // Empty name should cause validation error
					Mode:    SerialMode,
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		engine, err := NewEngine(config, registry)
		if err == nil {
			t.Error("Expected error for invalid layer")
		}
		if engine != nil {
			t.Error("Expected engine to be nil")
		}
	})
}

func TestEngineWithLogger(t *testing.T) {
	config := &Config{
		Name: "test-dag",
		Layers: []LayerConfig{
			{
				Name: "layer1",
				Mode: SerialMode,
				Components: []ComponentConfig{
					{Name: "comp1", Type: "test-type", Enabled: true},
				},
				Enabled: true,
			},
		},
	}

	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	t.Run("Create engine with custom logger", func(t *testing.T) {
		customLogger := &MockLogger{}
		engine, err := NewEngine(config, registry, WithLogger(customLogger))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if engine == nil {
			t.Error("Expected engine to be non-nil")
		}
	})
}

func TestEngineWithErrorHandler(t *testing.T) {
	config := &Config{
		Name: "test-dag",
		Layers: []LayerConfig{
			{
				Name: "layer1",
				Mode: SerialMode,
				Components: []ComponentConfig{
					{Name: "comp1", Type: "test-type", Enabled: true},
				},
				Enabled: true,
			},
		},
	}

	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	t.Run("Create engine with custom error handler", func(t *testing.T) {
		customErrorHandler := &MockErrorHandler{}
		engine, err := NewEngine(config, registry, WithErrorHandler(customErrorHandler))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if engine == nil {
			t.Error("Expected engine to be non-nil")
		}
	})
}

func TestEngineWithMiddleware(t *testing.T) {
	config := &Config{
		Name: "test-dag",
		Layers: []LayerConfig{
			{
				Name: "layer1",
				Mode: SerialMode,
				Components: []ComponentConfig{
					{Name: "comp1", Type: "test-type", Enabled: true},
				},
				Enabled: true,
			},
		},
	}

	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	t.Run("Create engine with middleware", func(t *testing.T) {
		middleware := &MockMiddleware{}
		engine, err := NewEngine(config, registry, WithMiddleware(middleware))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if engine == nil {
			t.Error("Expected engine to be non-nil")
		}
	})
}

func TestEngineGetters(t *testing.T) {
	config := &Config{
		Name: "test-dag",
		Layers: []LayerConfig{
			{
				Name: "layer1",
				Mode: SerialMode,
				Components: []ComponentConfig{
					{Name: "comp1", Type: "test-type", Enabled: true},
				},
				Enabled: true,
			},
		},
	}

	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	engine, _ := NewEngine(config, registry)

	t.Run("GetConfig", func(t *testing.T) {
		gotConfig := engine.GetConfig()
		if gotConfig != config {
			t.Error("Expected to get the same config")
		}
	})

	t.Run("GetLayers", func(t *testing.T) {
		layers := engine.GetLayers()
		if len(layers) != 1 {
			t.Errorf("Expected 1 layer, got %d", len(layers))
		}
	})

	t.Run("GetLayer existing", func(t *testing.T) {
		layer, exists := engine.GetLayer("layer1")
		if !exists {
			t.Error("Expected layer to exist")
		}
		if layer == nil {
			t.Error("Expected layer to be non-nil")
		}
		if layer.Name() != "layer1" {
			t.Errorf("Expected layer name to be 'layer1', got %s", layer.Name())
		}
	})

	t.Run("GetLayer non-existing", func(t *testing.T) {
		layer, exists := engine.GetLayer("non-existing")
		if exists {
			t.Error("Expected layer to not exist")
		}
		if layer != nil {
			t.Error("Expected layer to be nil")
		}
	})
}

func TestEngineValidate(t *testing.T) {
	t.Run("Validate valid engine", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{name: config.Name}, nil
			},
		}
		registry.Register(factory)

		engine, _ := NewEngine(config, registry)
		err := engine.Validate()
		if err != nil {
			t.Errorf("Expected no validation error, got %v", err)
		}
	})

	t.Run("Validate engine with invalid config", func(t *testing.T) {
		config := &Config{
			Name: "", // Empty name should cause validation error
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{name: config.Name}, nil
			},
		}
		registry.Register(factory)

		engine, _ := NewEngine(config, registry)
		err := engine.Validate()
		if err == nil {
			t.Error("Expected validation error")
		}
	})
}

func TestEngineExecute(t *testing.T) {
	t.Run("Execute simple DAG successfully", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{name: config.Name}, nil
			},
		}
		registry.Register(factory)

		engine, _ := NewEngine(config, registry)
		stats, err := engine.Execute(context.Background(), NewDataContext())

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stats == nil {
			t.Error("Expected stats to be non-nil")
		}
		if !stats.Success {
			t.Error("Expected execution to be successful")
		}
		if stats.LayersTotal != 1 {
			t.Errorf("Expected 1 total layer, got %d", stats.LayersTotal)
		}
		if stats.LayersSuccess != 1 {
			t.Errorf("Expected 1 successful layer, got %d", stats.LayersSuccess)
		}
	})

	t.Run("Execute DAG with component error", func(t *testing.T) {
		config := &Config{
			Name: "test-dag",
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
						return errors.New("component error")
					},
				}, nil
			},
		}
		registry.Register(factory)

		engine, _ := NewEngine(config, registry)
		stats, err := engine.Execute(context.Background(), NewDataContext())

		if err == nil {
			t.Error("Expected error")
		}
		if stats == nil {
			t.Error("Expected stats to be non-nil")
		}
		if stats.Success {
			t.Error("Expected execution to fail")
		}
		if stats.LayersTotal != 1 {
			t.Errorf("Expected 1 total layer, got %d", stats.LayersTotal)
		}
		if stats.LayersFailed != 1 {
			t.Errorf("Expected 1 failed layer, got %d", stats.LayersFailed)
		}
	})

	t.Run("Execute DAG with timeout", func(t *testing.T) {
		config := &Config{
			Name:    "test-dag",
			Timeout: 100 * time.Millisecond,
			Layers: []LayerConfig{
				{
					Name: "layer1",
					Mode: SerialMode,
					Components: []ComponentConfig{
						{Name: "comp1", Type: "test-type", Enabled: true},
					},
					Enabled: true,
				},
			},
		}

		registry := NewComponentRegistry()
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
					// Simulate long-running operation
						select {
						case <-time.After(200 * time.Millisecond):
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					},
				}, nil
			},
		}
		registry.Register(factory)

		engine, _ := NewEngine(config, registry)
		stats, err := engine.Execute(context.Background(), NewDataContext())

		if err == nil {
			t.Error("Expected timeout error")
		}
		if stats == nil {
			t.Error("Expected stats to be non-nil")
		}
		if stats.Success {
			t.Error("Expected execution to fail")
		}
	})
}

// MockLogger for testing
type MockLogger struct {
	logs []string
}

func (l *MockLogger) Debug(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "DEBUG: "+msg)
}

func (l *MockLogger) Info(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "INFO: "+msg)
}

func (l *MockLogger) Warn(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "WARN: "+msg)
}

func (l *MockLogger) Error(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "ERROR: "+msg)
}

// MockErrorHandler for testing
type MockErrorHandler struct {
	handledErrors []error
}

func (h *MockErrorHandler) HandleError(ctx context.Context, err error, component string, layer string) error {
	h.handledErrors = append(h.handledErrors, err)
	return nil
}

// MockMiddleware for testing
type MockMiddleware struct {
	beforeExecutionCalled bool
	afterExecutionCalled  bool
	beforeLayerCalled     bool
	afterLayerCalled      bool
}

func (m *MockMiddleware) BeforeExecution(ctx context.Context, config *Config) error {
	m.beforeExecutionCalled = true
	return nil
}

func (m *MockMiddleware) AfterExecution(ctx context.Context, stats *ExecutionStats) error {
	m.afterExecutionCalled = true
	return nil
}

func (m *MockMiddleware) BeforeLayer(ctx context.Context, layer *Layer) error {
	m.beforeLayerCalled = true
	return nil
}

func (m *MockMiddleware) AfterLayer(ctx context.Context, layer *Layer, stats *LayerStats) error {
	m.afterLayerCalled = true
	return nil
}
