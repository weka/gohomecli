package chart

import (
	"fmt"
	"strings"
)

var errConflictingKeys = fmt.Errorf("conflicting value overrides for key")

type yamlMap = map[string]interface{}

func writeMapEntryIfSet[T comparable](source yamlMap, key string, value T) error {
	var zero T

	if value == zero {
		return nil
	}

	return writeMapEntry(source, key, value)
}

func writeMapEntry(source yamlMap, key string, value interface{}) error {
	tokens := strings.Split(key, ".")
	currentMap := source
	for i, token := range tokens {
		if i == len(tokens)-1 {
			if _, ok := currentMap[token]; ok {
				return fmt.Errorf("%w: %s", errConflictingKeys, key)
			}

			currentMap[token] = value
			return nil
		}

		if _, ok := currentMap[token]; !ok {
			currentMap[token] = yamlMap{}
		}

		if nextMap, ok := currentMap[token].(yamlMap); !ok {
			return fmt.Errorf("%w: %s", errConflictingKeys, key)
		} else {
			currentMap = nextMap
		}
	}

	return nil
}

func mergeMaps(source yamlMap, overrides yamlMap, valuePath string) error {
	for key, value := range overrides {
		if _, ok := source[key]; !ok {
			source[key] = value
			continue
		}

		sourceMap, sourceMapOk := source[key].(yamlMap)
		overridesMap, overridesMapOk := overrides[key].(yamlMap)
		if sourceMapOk && overridesMapOk {
			subPath := fmt.Sprintf("%s.%s", valuePath, key)
			if err := mergeMaps(sourceMap, overridesMap, subPath); err != nil {
				return err
			}
		} else if sourceMapOk != overridesMapOk {
			return fmt.Errorf("%w: source=%T, overrides=%T", errConflictingKeys, source[key], overrides[key])
		} else {
			subPath := fmt.Sprintf("%s.%s", valuePath, key)
			return fmt.Errorf("%w: %s", errConflictingKeys, subPath)
		}
	}

	return nil
}
