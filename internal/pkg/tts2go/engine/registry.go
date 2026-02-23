package engine

import (
	"fmt"
	"sync"
)

type EngineFactory func(cfg EngineConfig) (Engine, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]EngineFactory)
)

func Register(name string, factory EngineFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if factory == nil {
		panic("engine: Register factory is nil")
	}
	if _, dup := registry[name]; dup {
		panic("engine: Register called twice for " + name)
	}
	registry[name] = factory
}

func New(name string, cfg EngineConfig) (Engine, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("engine: unknown backend %q (registered: %v)", name, ListBackends())
	}
	cfg.Backend = name
	return factory(cfg)
}

func ListBackends() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

func IsRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[name]
	return ok
}
