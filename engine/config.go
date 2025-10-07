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
    Extends     string                 `json:"extends,omitempty"`
}


// ConfigParser 配置解析器
type ConfigParser struct {
    envVarPattern *regexp.Regexp
    visitedExtends map[string]bool
}

// NewConfigParser 创建新的配置解析器
func NewConfigParser() *ConfigParser {
    return &ConfigParser{
        envVarPattern: regexp.MustCompile(`\$\{([^}]+)\}`),
        visitedExtends: make(map[string]bool),
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

    // 继承支持：如果存在 extends，则加载父配置并合并
    if config.Extends != "" {
        if p.visitedExtends[config.Extends] {
            return nil, &ConfigError{
                Type:    "extends_cycle_detected",
                Message: fmt.Sprintf("circular extends detected: %s", config.Extends),
            }
        }
        p.visitedExtends[config.Extends] = true

        parent, err := p.ParseFile(config.Extends)
        if err != nil {
            return nil, err
        }

        merged, err := p.mergeConfigs(parent, &config)
        if err != nil {
            return nil, err
        }

        // 验证合并后的配置
        if err := p.validateConfig(merged); err != nil {
            return nil, err
        }
        // 设置默认值
        p.setDefaults(merged)
        return merged, nil
    }

    // 验证配置
    if err := p.validateConfig(&config); err != nil {
        return nil, err
    }

    // 设置默认值
    p.setDefaults(&config)

    return &config, nil
}

// mergeConfigs 合并父子配置，实现继承与增删改
func (p *ConfigParser) mergeConfigs(parent, child *Config) (*Config, error) {
    // 基于父配置克隆副本
    base, err := parent.Clone()
    if err != nil {
        return nil, err
    }

    // 根字段：如果子配置提供非空值则覆盖
    if child.Name != "" {
        base.Name = child.Name
    }
    if child.Description != "" {
        base.Description = child.Description
    }
    if child.Version != "" {
        base.Version = child.Version
    }
    if child.Timeout > 0 {
        base.Timeout = child.Timeout
    }

    // 合并 Global 与 Metadata（子配置键覆盖父配置）
    if child.Global != nil {
        if base.Global == nil { base.Global = make(map[string]interface{}) }
        for k, v := range child.Global {
            base.Global[k] = v
        }
    }
    if child.Metadata != nil {
        if base.Metadata == nil { base.Metadata = make(map[string]string) }
        for k, v := range child.Metadata {
            base.Metadata[k] = v
        }
    }

    // 构建父层索引
    layerIdx := make(map[string]int)
    for i, l := range base.Layers {
        layerIdx[l.Name] = i
    }

    // 处理子层定义：新增、删除、修改
    for _, cl := range child.Layers {
        // 删除层：通过 remove 标记
        if cl.Remove {
            if idx, ok := layerIdx[cl.Name]; ok {
                // 删除该层
                base.Layers = append(base.Layers[:idx], base.Layers[idx+1:]...)
                // 更新索引
                layerIdx = make(map[string]int)
                for i, l := range base.Layers {
                    layerIdx[l.Name] = i
                }
            }
            continue
        }

        if idx, ok := layerIdx[cl.Name]; ok {
            // 修改/覆盖层
            bl := base.Layers[idx]
            if cl.Mode != "" { bl.Mode = cl.Mode }
            if cl.Timeout > 0 { bl.Timeout = cl.Timeout }
            // Enabled：仅在子为 true 时覆盖，避免未显式提供导致覆盖为 false
            if cl.Enabled {
                bl.Enabled = true
            }
            if cl.Parallel > 0 { bl.Parallel = cl.Parallel }
            if len(cl.Dependencies) > 0 { bl.Dependencies = cl.Dependencies }

            // 组件合并
            compIdx := make(map[string]int)
            for i, c := range bl.Components { compIdx[c.Name] = i }
            for _, cc := range cl.Components {
                if cc.Remove {
                    if cidx, ok := compIdx[cc.Name]; ok {
                        bl.Components = append(bl.Components[:cidx], bl.Components[cidx+1:]...)
                        // rebuild index
                        compIdx = make(map[string]int)
                        for i, c := range bl.Components { compIdx[c.Name] = i }
                    }
                    continue
                }
                if cidx, ok := compIdx[cc.Name]; ok {
                    bc := bl.Components[cidx]
                    if cc.Type != "" { bc.Type = cc.Type }
                    if cc.Timeout > 0 { bc.Timeout = cc.Timeout }
                    // 仅在子为 true 时覆盖 enabled，避免未显式提供导致覆盖为 false
                    if cc.Enabled {
                        bc.Enabled = true
                    }
                    // 仅在子为 true 时覆盖 critical（如需关闭可通过 remove 删除）
                    if cc.Critical {
                        bc.Critical = true
                    }
                    if len(cc.Dependencies) > 0 { bc.Dependencies = cc.Dependencies }
                    // 合并 config（子覆盖父）
                    if cc.Config != nil {
                        if bc.Config == nil { bc.Config = make(map[string]interface{}) }
                        for k, v := range cc.Config { bc.Config[k] = v }
                    }
                    // 覆盖 retry（如果提供）
                    if cc.Retry != nil { bc.Retry = cc.Retry }
                    bl.Components[cidx] = bc
                } else {
                    // 新增组件
                    bl.Components = append(bl.Components, cc)
                }
            }
            base.Layers[idx] = bl
        } else {
            // 新增层
            base.Layers = append(base.Layers, cl)
            layerIdx[cl.Name] = len(base.Layers) - 1
        }
    }

    return base, nil
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
