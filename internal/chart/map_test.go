package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	t.Run("should merge maps", func(t *testing.T) {
		source := yamlMap{
			"key2": yamlMap{
				"key3": "value3",
			},
		}

		overrides := yamlMap{
			"key1": "value1",
			"key2": yamlMap{
				"key2": "value3",
			},
		}

		if err := mergeMaps(source, overrides, ""); err != nil {
			t.Errorf("mergeMaps() error = %v", err)
		}

		expected := yamlMap{
			"key1": "value1",
			"key2": yamlMap{
				"key2": "value3",
				"key3": "value3",
			},
		}

		assert.Equal(t, expected, source)
	})

	t.Run("conflict on overwrite", func(t *testing.T) {
		source := yamlMap{"key1": "value1"}
		overrides := yamlMap{"key1": "value2"}
		err := mergeMaps(source, overrides, "")
		assert.ErrorIs(t, err, errConflictingKeys)
	})

	t.Run("non-map conflict", func(t *testing.T) {
		source := yamlMap{"key1": "value1"}
		overrides := yamlMap{"key1": yamlMap{"key2": "value2"}}

		err := mergeMaps(source, overrides, "")
		assert.ErrorIs(t, err, errConflictingKeys)
	})
}

func TestWriteMapEntry(t *testing.T) {
	t.Run("should write entries", func(t *testing.T) {
		source := yamlMap{"key1": "value1", "key2": "value2"}
		err := writeMapEntry(source, "key3", "value3")
		if err != nil {
			t.Errorf("writeMapEntry() error = %v", err)
		}

		expected := yamlMap{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		assert.Equal(t, expected, source)
	})

	t.Run("conflicting keys", func(t *testing.T) {
		source := yamlMap{
			"key1": "value1",
			"key2": "value2",
		}

		key := "key2"
		value := "value3"

		err := writeMapEntry(source, key, value)
		assert.ErrorIs(t, err, errConflictingKeys)

		expected := yamlMap{
			"key1": "value1",
			"key2": "value2",
		}

		assert.Equal(t, expected, source)
	})

	t.Run("conflicting types", func(t *testing.T) {
		source := yamlMap{
			"key1": "value1",
			"key2": "value2",
		}

		err := writeMapEntry(source, "key2", yamlMap{"key4": "value4"})
		assert.ErrorIs(t, err, errConflictingKeys)

		expected := yamlMap{
			"key1": "value1",
			"key2": "value2",
		}

		assert.Equal(t, expected, source)
	})
}
