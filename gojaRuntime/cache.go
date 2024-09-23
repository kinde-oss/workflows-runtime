package goja_runtime

import (
	"sync"

	"github.com/dop251/goja"
)

type gojaCache struct {
	cache map[string]*goja.Program
	lock  sync.Mutex
}

func (cache *gojaCache) cacheProgram(key string, loader func() (*goja.Program, error)) (*goja.Program, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if program, ok := cache.cache[key]; ok {
		return program, nil
	}

	program, err := loader()
	if err != nil {
		return nil, err
	}
	cache.cache[key] = program

	return program, nil
}
