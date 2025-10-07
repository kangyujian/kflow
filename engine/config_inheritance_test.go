package engine

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

func writeFile(t *testing.T, dir, name, content string) string {
    t.Helper()
    p := filepath.Join(dir, name)
    if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
        t.Fatalf("write file %s failed: %v", p, err)
    }
    return p
}

func findLayer(cfg *Config, name string) *LayerConfig {
    for i := range cfg.Layers {
        if cfg.Layers[i].Name == name {
            return &cfg.Layers[i]
        }
    }
    return nil
}

func findComponent(l *LayerConfig, name string) *ComponentConfig {
    for i := range l.Components {
        if l.Components[i].Name == name {
            return &l.Components[i]
        }
    }
    return nil
}

func TestConfigExtendsMerge_AddDeletePatch(t *testing.T) {
    dir := t.TempDir()

    parent := `{
        "name": "workflow-a",
        "layers": [
          {"name": "L1", "mode": "serial", "components": [
            {"name": "C1", "type": "Reader", "config": {"path": "a.txt"}, "enabled": true}
          ]},
          {"name": "L2", "mode": "parallel", "parallel": 4, "components": [
            {"name": "C2", "type": "Processor", "config": {"threshold": 0.8},
             "retry": {"max_retries": 2, "delay": 1000000000, "backoff": 1.5}}
          ]}
        ]
    }`
    child := `{
        "extends": "PARENT",
        "name": "workflow-b",
        "layers": [
          {"name": "L1", "remove": true},
          {"name": "L2", "mode": "parallel", "parallel": 8, "components": [
            {"name": "C2", "config": {"threshold": 0.9},
             "retry": {"max_retries": 3, "delay": 2000000000, "backoff": 2.0}},
            {"name": "C_new", "type": "Writer", "config": {"out": "b.txt"}}
          ]},
          {"name": "L3", "mode": "serial", "components": [
            {"name": "C3", "type": "X", "config": {"k": "v"}}
          ]}
        ]
    }`

    parentPath := writeFile(t, dir, "parent.json", parent)
    childPath := writeFile(t, dir, "child.json", child)

    // inject real parent path into child content
    b, err := os.ReadFile(childPath)
    if err != nil { t.Fatalf("read child: %v", err) }
    // Replace placeholder with actual parent path
    childContent := strings.ReplaceAll(string(b), "\"PARENT\"", "\""+parentPath+"\"")
    if err := os.WriteFile(childPath, []byte(childContent), 0o644); err != nil {
        t.Fatalf("rewrite child with path: %v", err)
    }

    parser := NewConfigParser()
    cfg, err := parser.ParseFile(childPath)
    if err != nil {
        t.Fatalf("ParseFile(child) failed: %v", err)
    }

    if cfg.Name != "workflow-b" {
        t.Fatalf("name not overridden, got %s", cfg.Name)
    }

    // L1 removed
    if l := findLayer(cfg, "L1"); l != nil {
        t.Fatalf("L1 should be removed")
    }
    // L2 patched
    l2 := findLayer(cfg, "L2")
    if l2 == nil { t.Fatalf("L2 not found") }
    if l2.Parallel != 8 { t.Fatalf("L2.parallel expected 8, got %d", l2.Parallel) }
    // C2 updated
    c2 := findComponent(l2, "C2")
    if c2 == nil { t.Fatalf("C2 not found") }
    if c2.Config == nil || c2.Config["threshold"] != 0.9 {
        t.Fatalf("C2.config.threshold expected 0.9, got %v", c2.Config["threshold"])
    }
    if c2.Retry == nil || c2.Retry.MaxRetries != 3 || c2.Retry.Delay != 2000000000 || c2.Retry.Backoff != 2.0 {
        t.Fatalf("C2.retry not overridden correctly: %+v", c2.Retry)
    }
    // C_new added
    if cnew := findComponent(l2, "C_new"); cnew == nil {
        t.Fatalf("C_new should be added")
    }
    // L3 added
    if l := findLayer(cfg, "L3"); l == nil {
        t.Fatalf("L3 should be added")
    }
}

func TestConfigExtendsCycleDetection(t *testing.T) {
    dir := t.TempDir()
    aPath := filepath.Join(dir, "a.json")
    bPath := filepath.Join(dir, "b.json")
    a := `{"name":"A","extends":"` + bPath + `","layers":[]}`
    b := `{"name":"B","extends":"` + aPath + `","layers":[]}`
    if err := os.WriteFile(aPath, []byte(a), 0o644); err != nil { t.Fatalf("write a: %v", err) }
    if err := os.WriteFile(bPath, []byte(b), 0o644); err != nil { t.Fatalf("write b: %v", err) }

    parser := NewConfigParser()
    _, err := parser.ParseFile(aPath)
    if err == nil {
        t.Fatalf("expected cycle detection error, got nil")
    }
}

func TestDefaultComponentTimeoutAndRemove(t *testing.T) {
    parser := NewConfigParser()
    // parent with two components, child removes one; also missing timeout should default to 30s
    parent := `{
      "name": "base",
      "layers": [
        { "name": "L1", "mode": "serial", "components": [
          {"name": "C1", "type": "X", "config": {"a": 1}},
          {"name": "C2", "type": "Y", "config": {"b": 2}}
        ]}
      ]
    }`
    child := `{
      "extends": "MEM",
      "name": "derived",
      "layers": [
        { "name": "L1", "components": [
          {"name": "C1", "remove": true}
        ]}
      ]
    }`

    dir := t.TempDir()
    parentPath := writeFile(t, dir, "p.json", parent)
    childPath := writeFile(t, dir, "c.json", child)
    cb, _ := os.ReadFile(childPath)
    cc := strings.ReplaceAll(string(cb), "\"MEM\"", "\""+parentPath+"\"")
    if err := os.WriteFile(childPath, []byte(cc), 0o644); err != nil { t.Fatalf("rewrite child: %v", err) }

    cfg, err := parser.ParseFile(childPath)
    if err != nil { t.Fatalf("parse child: %v", err) }
    l1 := findLayer(cfg, "L1")
    if l1 == nil { t.Fatalf("L1 not found") }
    if findComponent(l1, "C1") != nil { t.Fatalf("C1 should be removed") }
    c2 := findComponent(l1, "C2")
    if c2 == nil { t.Fatalf("C2 not found") }
    if c2.Timeout != 30*time.Second { t.Fatalf("C2 timeout default expected 30s, got %v", c2.Timeout) }
}