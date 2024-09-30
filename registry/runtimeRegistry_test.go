package runtime_registry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBindingMarshalling(t *testing.T) {
	settings := BindingSettings{
		Settings: map[string]interface{}{
			"key": "value",
		},
	}
	marshalled, err := json.Marshal(settings)
	assert := assert.New(t)

	assert.Nil(err)
	json.Unmarshal(marshalled, &settings)
	assert.Equal("value", settings.Settings["key"])
}
