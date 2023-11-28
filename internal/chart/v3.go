package chart

var valuesGeneratorV3 *yamlGenerator

func configureIngress(configuration *Configuration) (yamlMap, error) {
	cfg := make(yamlMap)

	if configuration.Host != "" {
		if err := writeMapEntry(cfg, "ingress.host", configuration.Host); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func init() {
	valuesGeneratorV3 = &yamlGenerator{
		visitors: map[string]configVisitor{},
	}

	valuesGeneratorV3.AddVisitor("ingress", configureIngress)
}

func generateValuesV3(configuration *Configuration) (map[string]interface{}, error) {
	return valuesGeneratorV3.Generate(configuration)
}
