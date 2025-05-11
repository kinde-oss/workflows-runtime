package goja_runtime

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

func TestVmPanicHandling(t *testing.T) {

	result := ""
	err := asyncRun(context.Background(), func(ctx context.Context) error {
		vm := goja.New()
		go func() {
			time.Sleep(1 * time.Second)
			vm.Interrupt("test interrupt") // kill script after 1 second
		}()
		_, err := vm.RunString("while (true) {}") //infinite loop
		defer func() {
			result = "test after"
		}()
		return err
	})
	assert.Error(t, err)
	assert.Equal(t, "test after", result)
}
