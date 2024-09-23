package url

import (
	_ "embed"
	"testing"

	"github.com/dop251/goja"
	"github.com/kinde-oss/workflows-runtime/gojaRuntime/gojaRuntime/require"
)

func createVM() *goja.Runtime {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	Enable(vm)
	return vm
}

func TestURLSearchParams(t *testing.T) {
	vm := createVM()

	if c := vm.Get("URLSearchParams"); c == nil {
		t.Fatal("URLSearchParams not found")
	}

	script := `const params = new URLSearchParams();`

	if _, err := vm.RunString(script); err != nil {
		t.Fatal("Failed to process url script.", err)
	}
}
