package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kangyujian/kflow/engine"
)

func main() {
	// 创建组件注册表
	registry := engine.NewComponentRegistry()

	// 注册组件工厂
	registerComponentFactories(registry)

	// 从配置文件加载
	parser := engine.NewConfigParser()
	var config *engine.Config
	var err error
	for _, p := range []string{"workflow.json", "example/basic/workflow.json"} {
		if _, statErr := os.Stat(p); statErr == nil {
			config, err = parser.ParseFile(p)
			break
		}
	}
	if config == nil && err == nil {
		// 兜底尝试默认路径
		config, err = parser.ParseFile("workflow.json")
	}
	if err != nil || config == nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 创建引擎
	kflowEngine, err := engine.NewEngine(config, registry)
	if err != nil {
		log.Fatalf("创建引擎失败: %v", err)
	}

	// 创建并发安全的数据上下文
	dataCtx := engine.NewDataContext()
	ctx := context.Background()

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 执行工作流
	fmt.Println("开始执行工作流...")
	start := time.Now()

	_, err = kflowEngine.Execute(ctx, dataCtx)
	if err != nil {
		log.Fatalf("执行工作流失败: %v", err)
	}

	fmt.Printf("工作流执行完成，耗时: %v\n", time.Since(start))

	// 打印结果快照
	fmt.Println("\n执行结果:")
	for k, v := range dataCtx.Snapshot() {
		fmt.Printf("%s: %v\n", k, v)
	}
}

// 定义组件工厂类型
type fileReaderFactory struct{}

func (f *fileReaderFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	filePath, _ := config.Config["file_path"].(string)
	encoding, _ := config.Config["encoding"].(string)

	return &FileReaderComponent{
		name:     config.Name,
		filePath: filePath,
		encoding: encoding,
		isCore:   config.Critical,
	}, nil
}

func (f *fileReaderFactory) GetType() string {
	return "file_reader"
}

type configReaderFactory struct{}

func (f *configReaderFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	configPath, _ := config.Config["config_path"].(string)

	return &ConfigReaderComponent{
		name:       config.Name,
		configPath: configPath,
		isCore:     config.Critical,
	}, nil
}

func (f *configReaderFactory) GetType() string {
	return "config_reader"
}

type transformerFactory struct{}

func (f *transformerFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	var operations []string
	if ops, ok := config.Config["operations"].([]interface{}); ok {
		for _, op := range ops {
			if strOp, ok := op.(string); ok {
				operations = append(operations, strOp)
			}
		}
	}

	return &TransformerComponent{
		name:       config.Name,
		operations: operations,
		isCore:     config.Critical,
	}, nil
}

func (f *transformerFactory) GetType() string {
	return "transformer"
}

type validatorFactory struct{}

func (f *validatorFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	var rules []string
	if r, ok := config.Config["rules"].([]interface{}); ok {
		for _, rule := range r {
			if strRule, ok := rule.(string); ok {
				rules = append(rules, strRule)
			}
		}
	}

	return &ValidatorComponent{
		name:   config.Name,
		rules:  rules,
		isCore: config.Critical,
	}, nil
}

func (f *validatorFactory) GetType() string {
	return "validator"
}

type fileWriterFactory struct{}

func (f *fileWriterFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	outputPath, _ := config.Config["output_path"].(string)
	appendMode, _ := config.Config["append"].(bool)

	return &FileWriterComponent{
		name:       config.Name,
		outputPath: outputPath,
		append:     appendMode,
		isCore:     config.Critical,
	}, nil
}

func (f *fileWriterFactory) GetType() string {
	return "file_writer"
}

type loggerFactory struct{}

func (f *loggerFactory) Create(config engine.ComponentConfig) (engine.Component, error) {
	level, _ := config.Config["level"].(string)
	message, _ := config.Config["message"].(string)

	return &LoggerComponent{
		name:    config.Name,
		level:   level,
		message: message,
		isCore:  config.Critical,
	}, nil
}

func (f *loggerFactory) GetType() string {
	return "logger"
}

// 注册组件工厂
func registerComponentFactories(registry *engine.ComponentRegistry) {
	registry.Register(&fileReaderFactory{})
	registry.Register(&configReaderFactory{})
	registry.Register(&transformerFactory{})
	registry.Register(&validatorFactory{})
	registry.Register(&fileWriterFactory{})
	registry.Register(&loggerFactory{})
}