package engine

import (
	"sync"
)

// DataContext 定义组件间共享数据的并发安全接口
// 通过细粒度的读写锁保证并发读写的安全性
// 提供常用的便捷方法与快照功能用于调试或输出
// 注意：Snapshot 返回的是当前数据的浅拷贝
// 对于引用类型的值（如 map/slice），调用方仍需自行保证内部数据的安全使用
type DataContext interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	GetString(key string) (string, bool)
	Delete(key string)
	Has(key string) bool
	Snapshot() map[string]interface{}
}

// defaultDataContext 是 DataContext 的默认实现
// 使用 RWMutex 保护内部 map 的并发访问
type defaultDataContext struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewDataContext 创建一个空的并发安全数据上下文
func NewDataContext() DataContext {
	return &defaultDataContext{data: make(map[string]interface{})}
}

// NewDataContextWith 初始化一个并发安全数据上下文
func NewDataContextWith(initial map[string]interface{}) DataContext {
	ctx := &defaultDataContext{data: make(map[string]interface{}, len(initial))}
	for k, v := range initial {
		ctx.data[k] = v
	}
	return ctx
}

func (c *defaultDataContext) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *defaultDataContext) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *defaultDataContext) GetString(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.data[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return s, true
		}
	}
	return "", false
}

func (c *defaultDataContext) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *defaultDataContext) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[key]
	return ok
}

func (c *defaultDataContext) Snapshot() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copy := make(map[string]interface{}, len(c.data))
	for k, v := range c.data {
		copy[k] = v
	}
	return copy
}