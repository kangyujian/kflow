package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestExecutionMode(t *testing.T) {
	t.Run("ExecutionMode constants", func(t *testing.T) {
		if SerialMode != "serial" {
			t.Errorf("Expected SerialMode to be 'serial', got %s", SerialMode)
		}
		if ParallelMode != "parallel" {
			t.Errorf("Expected ParallelMode to be 'parallel', got %s", ParallelMode)
		}
		if AsyncMode != "async" {
			t.Errorf("Expected AsyncMode to be 'async', got %s", AsyncMode)
		}
	})
}

func TestLayerConfig(t *testing.T) {
	t.Run("LayerConfig creation", func(t *testing.T) {
		config := LayerConfig{
			Name:         "test-layer",
			Mode:         SerialMode,
			Components:   []ComponentConfig{},
			Timeout:      30 * time.Second,
			Dependencies: []string{"dep1"},
			Enabled:      true,
			Parallel:     2,
		}

		if config.Name != "test-layer" {
			t.Errorf("Expected Name to be 'test-layer', got %s", config.Name)
		}
		if config.Mode != SerialMode {
			t.Errorf("Expected Mode to be SerialMode, got %s", config.Mode)
		}
		if config.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout to be 30s, got %v", config.Timeout)
		}
		if !config.Enabled {
			t.Error("Expected Enabled to be true")
		}
		if config.Parallel != 2 {
			t.Errorf("Expected Parallel to be 2, got %d", config.Parallel)
		}
	})
}

func TestNewLayer(t *testing.T) {
	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	t.Run("Create layer with valid components", func(t *testing.T) {
		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, err := NewLayer(config, registry)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if layer == nil {
			t.Error("Expected layer to be non-nil")
		}
		if len(layer.Components()) != 2 {
			t.Errorf("Expected 2 components, got %d", len(layer.Components()))
		}
	})

	t.Run("Create layer with disabled components", func(t *testing.T) {
		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: false}, // disabled
			},
			Enabled: true,
		}

		layer, err := NewLayer(config, registry)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(layer.Components()) != 1 {
			t.Errorf("Expected 1 component (disabled one should be skipped), got %d", len(layer.Components()))
		}
	})

	t.Run("Create layer with invalid component type", func(t *testing.T) {
		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "unknown-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, err := NewLayer(config, registry)
		if err == nil {
			t.Error("Expected error for unknown component type")
		}
		if layer != nil {
			t.Error("Expected layer to be nil when component creation fails")
		}
	})
}

func TestLayerMethods(t *testing.T) {
	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockComponent{name: config.Name}, nil
		},
	}
	registry.Register(factory)

	config := LayerConfig{
		Name: "test-layer",
		Mode: ParallelMode,
		Components: []ComponentConfig{
			{Name: "comp1", Type: "test-type", Enabled: true},
		},
		Enabled: true,
	}

	layer, _ := NewLayer(config, registry)

	t.Run("Name method", func(t *testing.T) {
		if layer.Name() != "test-layer" {
			t.Errorf("Expected name to be 'test-layer', got %s", layer.Name())
		}
	})

	t.Run("Mode method", func(t *testing.T) {
		if layer.Mode() != ParallelMode {
			t.Errorf("Expected mode to be ParallelMode, got %s", layer.Mode())
		}
	})

	t.Run("Components method", func(t *testing.T) {
		components := layer.Components()
		if len(components) != 1 {
			t.Errorf("Expected 1 component, got %d", len(components))
		}
		if components[0].Name() != "comp1" {
			t.Errorf("Expected component name to be 'comp1', got %s", components[0].Name())
		}
	})
}

func TestLayerExecute(t *testing.T) {
	registry := NewComponentRegistry()

	t.Run("Execute serial mode", func(t *testing.T) {
		var executionOrder []string
		var mu sync.Mutex

		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
						mu.Lock()
						executionOrder = append(executionOrder, config.Name)
						mu.Unlock()
						time.Sleep(10 * time.Millisecond) // Small delay to test ordering
						return nil
					},
				}, nil
			},
		}
		registry.Register(factory)

		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
				{Name: "comp3", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Execute(context.Background(), NewDataContext())

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// In serial mode, components should execute in order
		expectedOrder := []string{"comp1", "comp2", "comp3"}
		if len(executionOrder) != len(expectedOrder) {
			t.Errorf("Expected %d executions, got %d", len(expectedOrder), len(executionOrder))
		}
		for i, expected := range expectedOrder {
			if i >= len(executionOrder) || executionOrder[i] != expected {
				t.Errorf("Expected execution order %v, got %v", expectedOrder, executionOrder)
				break
			}
		}
	})

	t.Run("Execute parallel mode", func(t *testing.T) {
		var executionCount int32
		var mu sync.Mutex

		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
					mu.Lock()
						executionCount++
						mu.Unlock()
						time.Sleep(50 * time.Millisecond) // Simulate work
						return nil
					},
				}, nil
			},
		}
		registry.Register(factory)

		config := LayerConfig{
			Name: "test-layer",
			Mode: ParallelMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
				{Name: "comp3", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		start := time.Now()
		err := layer.Execute(context.Background(), NewDataContext())
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Parallel execution should be faster than serial
		if duration > 100*time.Millisecond {
			t.Errorf("Parallel execution took too long: %v", duration)
		}

		if executionCount != 3 {
			t.Errorf("Expected 3 executions, got %d", executionCount)
		}
	})

	t.Run("Execute async mode", func(t *testing.T) {
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
					time.Sleep(100 * time.Millisecond) // Simulate work
						return nil
					},
				}, nil
			},
		}
		registry.Register(factory)

		config := LayerConfig{
			Name: "test-layer",
			Mode: AsyncMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		start := time.Now()
		err := layer.Execute(context.Background(), NewDataContext())
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Async execution should return immediately
		if duration > 50*time.Millisecond {
			t.Errorf("Async execution took too long: %v", duration)
		}
	})

	t.Run("Execute with component error in serial mode", func(t *testing.T) {
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
					if config.Name == "comp2" {
							return errors.New("component error")
						}
						return nil
					},
				}, nil
			},
		}
		registry.Register(factory)

		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
				{Name: "comp3", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Execute(context.Background(), NewDataContext())

		if err == nil {
			t.Error("Expected error from failing component")
		}
	})

	t.Run("Execute with timeout", func(t *testing.T) {
		factory := &MockComponentFactory{
			componentType: "test-type",
			createFunc: func(config ComponentConfig) (Component, error) {
				return &MockComponent{
					name: config.Name,
					executeFunc: func(ctx context.Context, data DataContext) error {
					// Simulate long-running operation that respects context cancellation
						select {
						case <-time.After(200 * time.Millisecond): // Longer than timeout
							return nil
						case <-ctx.Done():
							return ctx.Err() // Return context error (deadline exceeded)
						}
					},
				}, nil
			},
		}
		registry.Register(factory)

		config := LayerConfig{
			Name:    "test-layer",
			Mode:    SerialMode,
			Timeout: 100 * time.Millisecond, // Short timeout
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Execute(context.Background(), NewDataContext())

		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

func TestLayerValidate(t *testing.T) {
	registry := NewComponentRegistry()
	factory := &MockComponentFactory{
		componentType: "test-type",
		createFunc: func(config ComponentConfig) (Component, error) {
			return &MockValidatableComponent{
				MockComponent: MockComponent{name: config.Name},
				validateFunc: func() error {
					if config.Name == "invalid-comp" {
						return errors.New("validation error")
					}
					return nil
				},
			}, nil
		},
	}
	registry.Register(factory)

	t.Run("Validate layer with valid components", func(t *testing.T) {
		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "comp2", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Validate()

		if err != nil {
			t.Errorf("Expected no validation error, got %v", err)
		}
	})

	t.Run("Validate layer with invalid component", func(t *testing.T) {
		config := LayerConfig{
			Name: "test-layer",
			Mode: SerialMode,
			Components: []ComponentConfig{
				{Name: "comp1", Type: "test-type", Enabled: true},
				{Name: "invalid-comp", Type: "test-type", Enabled: true},
			},
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Validate()

		if err == nil {
			t.Error("Expected validation error")
		}
	})

	t.Run("Validate layer with empty name", func(t *testing.T) {
		config := LayerConfig{
			Name:    "", // Empty name
			Mode:    SerialMode,
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Validate()

		if err == nil {
			t.Error("Expected validation error for empty name")
		}
	})

	t.Run("Validate layer with invalid mode", func(t *testing.T) {
		config := LayerConfig{
			Name:    "test-layer",
			Mode:    ExecutionMode("invalid"), // Invalid mode
			Enabled: true,
		}

		layer, _ := NewLayer(config, registry)
		err := layer.Validate()

		if err == nil {
			t.Error("Expected validation error for invalid mode")
		}
	})
}