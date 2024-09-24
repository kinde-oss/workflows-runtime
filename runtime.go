package workflows_runtime

import (
	_ "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	registry "github.com/kinde-oss/workflows-runtime/registry"
)

func GetRuntime(name string) (registry.Runner, error) {
	return registry.ResolveRuntime(name)
}
