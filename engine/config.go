package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// Config DAG 配置
type Config struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Layers      []LayerConfig          `json:"layers"`
	Global      map[string]interface{} `json:"global,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// ConfigParser 配置解析器
type ConfigParser struct {
	envVarPattern *regexp.Regexp
}

// NewConfigParser 创建新的配置解析器
func NewConfigParser() *ConfigParser {
	return &ConfigParser{
		envVarPattern: regexp.MustCompile(`\$\{([^}]+)\}`),
	}
}

// ParseFile 从文件解析配置
func (p *ConfigParser) ParseFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, &ConfigError{
			Type:    "file_open_failed",
			Message: fmt.Sprintf("failed to open config file: %v", err),
			Cause:   err,
		}
	}
	defer file.Close()

	return p.Parse(file)
}

// Parse 从 Reader 解析配置
func (p *ConfigParser) Parse(reader io.Reader) (*Config, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, &ConfigError{
			Type:    "read_failed",
			Message: fmt.Sprintf("failed to read config data: %v", err),
			Cause:   err,
		}
	}

	return p.ParseBytes(data)
}

// ParseBytes 从字节数组解析配置
func (p *ConfigParser) ParseBytes(data []byte) (*Config, error) {
	// 替换环境变量
	configStr := p.replaceEnvVars(string(data))

	var config Config
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, &ConfigError{
			Type:    "json_unmarshal_failed",
			Message: fmt.Sprintf("failed to unmarshal JSON config: %v", err),
			Cause:   err,
		}
	}

	// 验证配置
	if err := p.validateConfig(&config); err != nil {
		return nil, err
	}

	// 设置默认值
	p.setDefaults(&config)

	return &config, nil
}

// replaceEnvVars 替换配置中的环境变量
func (p *ConfigParser) replaceEnvVars(configStr string) string {
	return p.envVarPattern.ReplaceAllStringFunc(configStr, func(match string) string {
		// 提取变量名 (去掉 ${ 和 })
		varName := match[2 : len(match)-1]

		// 支持默认值语法: ${VAR_NAME:default_value}
		parts := strings.SplitN(varName, ":", 2)
		envVar := parts[0]
		defaultValue := ""

		if len(parts) > 1 {
			defaultValue = parts[1]
		}

		// 获取环境变量值
		if value := os.Getenv(envVar); value != "" {
			return value
		}

		return defaultValue
	})
}

// validateConfig 验证配置
func (p *ConfigParser) validateConfig(config *Config) error {
	if config.Name == "" {
		return &ValidationError{
			Field:   "name",
			Value:   config.Name,
			Message: "config name cannot be empty",
		}
	}

	if len(config.Layers) == 0 {
		return &ValidationError{
			Field:   "layers",
			Value:   config.Layers,
			Message: "at least one layer must be defined",
		}
	}

	// 验证层级名称唯一性
	layerNames := make(map[string]bool)
	for i, layer := range config.Layers {
		if layer.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("layers[%d].name", i),
				Value:   layer.Name,
				Message: "layer name cannot be empty",
			}
		}

		if layerNames[layer.Name] {
			return &ValidationError{
				Field:   fmt.Sprintf("layers[%d].name", i),
				Value:   layer.Name,
				Message: fmt.Sprintf("duplicate layer name: %s", layer.Name),
			}
		}
		layerNames[layer.Name] = true

		// 验证层级配置
		if err := p.validateLayerConfig(&layer, i); err != nil {
			return err
		}
	}

	// 验证层级依赖
	if err := p.validateLayerDependencies(config.Layers); err != nil {
		return err
	}

	return nil
}

// validateLayerConfig 验证层级配置
func (p *ConfigParser) validateLayerConfig(layer *LayerConfig, index int) error {
	// 验证执行模式
	validModes := map[ExecutionMode]bool{
		SerialMode:   true,
		ParallelMode: true,
		AsyncMode:    true,
	}

	if layer.Mode == "" {
		layer.Mode = SerialMode // 默认串行模式
	}

	if !validModes[layer.Mode] {
		return &ValidationError{
			Field:   fmt.Sprintf("layers[%d].mode", index),
			Value:   layer.Mode,
			Message: fmt.Sprintf("invalid execution mode: %s", layer.Mode),
		}
	}

	// 验证组件
	if len(layer.Components) == 0 {
		return &ValidationError{
			Field:   fmt.Sprintf("layers[%d].components", index),
			Value:   layer.Components,
			Message: "layer must have at least one component",
		}
	}

	// 验证组件名称唯一性
	componentNames := make(map[string]bool)
	for j, component := range layer.Components {
		if component.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("layers[%d].components[%d].name", index, j),
				Value:   component.Name,
				Message: "component name cannot be empty",
			}
		}

		if componentNames[component.Name] {
			return &ValidationError{
				Field:   fmt.Sprintf("layers[%d].components[%d].name", index, j),
				Value:   component.Name,
				Message: fmt.Sprintf("duplicate component name in layer: %s", component.Name),
			}
		}
		componentNames[component.Name] = true

		// 验证组件类型
		if component.Type == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("layers[%d].components[%d].type", index, j),
				Value:   component.Type,
				Message: "component type cannot be empty",
			}
		}
	}

	return nil
}

// validateLayerDependencies 验证层级依赖关系
func (p *ConfigParser) validateLayerDependencies(layers []LayerConfig) error {
	layerMap := make(map[string]int)
	for i, layer := range layers {
		layerMap[layer.Name] = i
	}

	for i, layer := range layers {
		for _, dep := range layer.Dependencies {
			depIndex, exists := layerMap[dep]
			if !exists {
				return &ValidationError{
					Field:   fmt.Sprintf("layers[%d].dependencies", i),
					Value:   dep,
					Message: fmt.Sprintf("dependency layer not found: %s", dep),
				}
			}

			// 依赖的层级必须在当前层级之前
			if depIndex >= i {
				return &ValidationError{
					Field:   fmt.Sprintf("layers[%d].dependencies", i),
					Value:   dep,
					Message: fmt.Sprintf("circular or forward dependency detected: %s", dep),
				}
			}
		}
	}

	return nil
}

// setDefaults 设置默认值
func (p *ConfigParser) setDefaults(config *Config) {
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	for i := range config.Layers {
		layer := &config.Layers[i]

		if layer.Mode == "" {
			layer.Mode = SerialMode
		}

		if layer.Enabled == false && layer.Mode != "" {
			layer.Enabled = true // 默认启用
		}

		for j := range layer.Components {
			component := &layer.Components[j]

			if component.Enabled == false && component.Type != "" {
				component.Enabled = true // 默认启用
			}

			if component.Timeout == 0 {
				component.Timeout = 30 * time.Second // 默认30秒超时
			}
		}
	}
}

// ToJSON 将配置转换为 JSON 字符串
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", &ConfigError{
			Type:    "json_marshal_failed",
			Message: fmt.Sprintf("failed to marshal config to JSON: %v", err),
			Cause:   err,
		}
	}
	return string(data), nil
}

// GetLayer 根据名称获取层级配置
func (c *Config) GetLayer(name string) (*LayerConfig, bool) {
	for i := range c.Layers {
		if c.Layers[i].Name == name {
			return &c.Layers[i], true
		}
	}
	return nil, false
}

// GetComponent 根据名称获取组件配置
func (c *Config) GetComponent(layerName, componentName string) (*ComponentConfig, bool) {
	layer, exists := c.GetLayer(layerName)
	if !exists {
		return nil, false
	}

	for i := range layer.Components {
		if layer.Components[i].Name == componentName {
			return &layer.Components[i], true
		}
	}
	return nil, false
}

// Clone 克隆配置
func (c *Config) Clone() (*Config, error) {
	jsonStr, err := c.ToJSON()
	if err != nil {
		return nil, err
	}

	parser := NewConfigParser()
	return parser.ParseBytes([]byte(jsonStr))
}
