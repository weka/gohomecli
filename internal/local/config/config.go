package config

import (
	"encoding/json"
	"fmt"
	"os"

	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("configuration")

func ReadV1(jsonConfig string, config *config_v1.Configuration) error {
	if jsonConfig == "" {
		return nil
	}

	var jsonConfigBytes []byte
	if _, err := os.Stat(jsonConfig); err == nil {
		logger.Debug().Str("path", jsonConfig).Msg("Reading JSON config from file")
		jsonConfigBytes, err = os.ReadFile(jsonConfig)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to read JSON config from file")
			return fmt.Errorf("failed to read JSON config from file: %w", err)
		}
	} else {
		logger.Debug().Msg("Using JSON object from command line")
		jsonConfigBytes = []byte(jsonConfig)
	}

	logger.Debug().Msg("Parsing JSON config")
	err := json.Unmarshal(jsonConfigBytes, config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse JSON config")
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return nil
}
