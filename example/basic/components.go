package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kangyujian/kflow/engine"
)

// FileReaderComponent 文件读取组件
type FileReaderComponent struct {
	name     string
	filePath string
	encoding string
	isCore   bool
	data     string
}

func (c *FileReaderComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("FileReader: 读取文件 %s\n", c.filePath)

	// 真实文件读取，支持在项目根或 example/basic 下运行
	paths := []string{c.filePath}
	if !filepath.IsAbs(c.filePath) {
		paths = append(paths, filepath.Join("example/basic", c.filePath))
	}

	var lastErr error
	for _, p := range paths {
		bytes, err := os.ReadFile(p)
		if err == nil {
			c.data = string(bytes)
			data.Set("file_data", c.data)
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("读取文件失败(尝试路径: %v): %w", paths, lastErr)
}

func (c *FileReaderComponent) Name() string {
	return c.name
}

// ConfigReaderComponent 配置读取组件
type ConfigReaderComponent struct {
	name       string
	configPath string
	isCore     bool
}

func (c *ConfigReaderComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("ConfigReader: 读取配置文件 %s\n", c.configPath)
	
	// 模拟配置读取
	config := map[string]string{
		"app_name": "KFlow Demo",
		"version":  "1.0.0",
		"env":      "development",
	}
	
	// 将配置存入上下文
	data.Set("config", config)
	
	return nil
}

func (c *ConfigReaderComponent) Name() string {
	return c.name
}

// TransformerComponent 数据转换组件
type TransformerComponent struct {
	name       string
	operations []string
	isCore     bool
}

func (c *TransformerComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("Transformer: 执行数据转换 %v\n", c.operations)
	
	// 从上下文获取数据
	fileData, ok := data.GetString("file_data")
	if !ok {
		return fmt.Errorf("无法获取文件数据")
	}
	
	// 执行转换操作
	result := fileData
	for _, op := range c.operations {
		switch op {
		case "uppercase":
			result = strings.ToUpper(result)
		case "trim":
			result = strings.TrimSpace(result)
		}
	}
	
	// 更新上下文中的数据
	data.Set("transformed_data", result)
	
	return nil
}

func (c *TransformerComponent) Name() string {
	return c.name
}

// ValidatorComponent 数据验证组件
type ValidatorComponent struct {
	name   string
	rules  []string
	isCore bool
}

func (c *ValidatorComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("Validator: 执行数据验证 %v\n", c.rules)
	
	// 从上下文获取数据
	transformedData, ok := data.GetString("transformed_data")
	if !ok {
		return fmt.Errorf("无法获取转换后的数据")
	}
	
	// 执行验证规则
	for _, rule := range c.rules {
		if rule == "not_empty" && len(transformedData) == 0 {
			return fmt.Errorf("数据不能为空")
		}
		
		if strings.HasPrefix(rule, "max_length:") {
			parts := strings.Split(rule, ":")
			if len(parts) != 2 {
				continue
			}
			
			var maxLen int
			fmt.Sscanf(parts[1], "%d", &maxLen)
			
			if len(transformedData) > maxLen {
				return fmt.Errorf("数据长度超过最大限制: %d", maxLen)
			}
		}
	}
	
	return nil
}

func (c *ValidatorComponent) Name() string {
	return c.name
}

// FileWriterComponent 文件写入组件
type FileWriterComponent struct {
	name       string
	outputPath string
	append     bool
	isCore     bool
}

func (c *FileWriterComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("FileWriter: 写入文件 %s\n", c.outputPath)

	// 从上下文获取数据
	transformedData, ok := data.GetString("transformed_data")
	if !ok {
		return fmt.Errorf("无法获取转换后的数据")
	}

	// 解析写入路径：优先使用给定路径，若相对路径在根目录不存在，则写入 example/basic 下
	target := c.outputPath
	if !filepath.IsAbs(target) {
		if _, err := os.Stat(target); err != nil {
			alt := filepath.Join("example/basic", target)
			target = alt
		}
	}

	// 真实文件写入
	if c.append {
		f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("打开文件失败: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(transformedData + "\n"); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	} else {
		if err := os.WriteFile(target, []byte(transformedData), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	return nil
}

func (c *FileWriterComponent) Name() string {
	return c.name
}

// LoggerComponent 日志组件
type LoggerComponent struct {
	name    string
	level   string
	message string
	isCore  bool
}

func (c *LoggerComponent) Execute(ctx context.Context, data engine.DataContext) error {
	fmt.Printf("[%s] %s: %s\n", c.level, time.Now().Format("2006-01-02 15:04:05"), c.message)
	return nil
}

func (c *LoggerComponent) Name() string {
	return c.name
}