package url

import (
	_ "embed"
	"testing"

	"github.com/dop251/goja"
	"github.com/kinde-oss/workflows-runtime/gojaRuntime/require"
)

func TestURLSearchParamsAppend(t *testing.T) {
	vm := createVM()

	script := `
		const params = new URLSearchParams();
		params.append("name", "value");
		params.get("name");
	`

	result, err := vm.RunString(script)
	if err != nil {
		t.Fatal("Failed to process url script.", err)
	}

	if result.String() != "value" {
		t.Fatalf("Expected 'value', got '%s'", result.String())
	}
}

func TestURLSearchParamsGetAll(t *testing.T) {
	vm := createVM()

	script := `
		const params = new URLSearchParams();
		params.append("name", "value1");
		params.append("name", "value2");
		params.getAll("name");
	`

	result, err := vm.RunString(script)
	if err != nil {
		t.Fatal("Failed to process url script.", err)
	}

	expected := `value1,value2`
	if result.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, result.String())
	}
}

func TestURLSearchParamsHas(t *testing.T) {
	vm := createVM()

	script := `
		const params = new URLSearchParams();
		params.append("name", "value");
		params.has("name");
	`

	result, err := vm.RunString(script)
	if err != nil {
		t.Fatal("Failed to process url script.", err)
	}

	if !result.ToBoolean() {
		t.Fatalf("Expected true, got false")
	}
}

func TestURLSearchParamsSet(t *testing.T) {
	vm := createVM()

	script := `
		const params = new URLSearchParams();
		params.append("name", "value1");
		params.set("name", "value2");
		params.get("name");
	`

	result, err := vm.RunString(script)
	if err != nil {
		t.Fatal("Failed to process url script.", err)
	}

	if result.String() != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", result.String())
	}
}

func TestURLSearchParamsSort(t *testing.T) {
	vm := createVM()

	script := `
		const params = new URLSearchParams();
		params.append("b", "value2");
		params.append("a", "value1");
		params.sort();
		params.toString();
	`

	result, err := vm.RunString(script)
	if err != nil {
		t.Fatal("Failed to process url script.", err)
	}

	expected := "a=value1&b=value2"
	if result.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, result.String())
	}
}
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
