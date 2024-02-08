package chart

import (
	"errors"
	"fmt"

	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
)

var (
	ErrMalformedConfiguration = fmt.Errorf("logic error, malformed configuration generated")
	ErrGenerationFailed       = fmt.Errorf("failed to generate configuration")
)

type (
	configVisitor = func(configuration *config_v1.Configuration) (yamlMap, error)
	yamlGenerator struct {
		visitors map[string]configVisitor
	}
)

func (g *yamlGenerator) MustAddVisitor(name string, visitor configVisitor) {
	if _, exist := g.visitors[name]; exist {
		panic(fmt.Sprintf("visitor with name %s already exists, unable to register", name))
	}

	g.visitors[name] = visitor
}

func (g *yamlGenerator) Generate(configuration *config_v1.Configuration) (yamlMap, error) {
	result := yamlMap{}

	for name, visitor := range g.visitors {
		visitorResult, err := visitor(configuration)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGenerationFailed, err)
		}

		if err = mergeMaps(result, visitorResult, ""); err != nil {
			if errors.Is(err, errConflictingKeys) {
				return nil, fmt.Errorf("%w: %s", ErrMalformedConfiguration, name)
			}

			return nil, err
		}
	}

	return result, nil
}
