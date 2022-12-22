package env

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pelletier/go-toml"
	"os"
)

type Aliases struct {
	FilePath    string
	initialized bool
	data        map[string]interface{}
}

func NewAliases() *Aliases {
	return &Aliases{
		FilePath:    AliasesFilePath,
		initialized: false,
		data:        make(map[string]interface{}),
	}
}

func (aliases *Aliases) Init() {
	if aliases.initialized {
		return
	}
	if _, err := os.Stat(aliases.FilePath); !(err != nil && os.IsNotExist(err)) {
		aliases.load()
		logger.Debug().Msg("Aliases loaded")
	} else {
		logger.Debug().Str("file", aliases.FilePath).Msg("Aliases file does not exist")
	}
	aliases.initialized = true
}

func (aliases *Aliases) Iter(f func(string, string)) {
	aliases.Init()
	for alias, clusterID := range aliases.data {
		f(alias, clusterID.(string))
	}
}

func (aliases *Aliases) Get(aliasOrClusterID string) (string, bool) {
	aliases.Init()
	clusterID, aliasExists := aliases.data[aliasOrClusterID]
	var clusterIDStr string
	if aliasExists {
		clusterIDStr = clusterID.(string)
	} else {
		clusterIDStr = aliasOrClusterID
	}
	return clusterIDStr, aliasExists
}

func (aliases *Aliases) Set(alias string, clusterID string, override bool) error {
	aliases.Init()
	for existingAlias, existingClusterID := range aliases.data {
		if existingAlias == alias {
			return fmt.Errorf("alias already exists: %s", alias)
		}
		existingClusterIDStr := existingClusterID.(string)
		if existingClusterIDStr == clusterID {
			return fmt.Errorf("cluster ID %s already aliased to \"%s\"", clusterID, existingAlias)
		}
	}
	_, exists := aliases.data[alias]
	if exists && !override {
		return fmt.Errorf("alias already exists: %s", alias)
	}
	aliases.data[alias] = clusterID
	aliases.save()
	return nil
}

func (aliases *Aliases) Remove(alias string) error {
	aliases.Init()
	_, exists := aliases.data[alias]
	if !exists {
		return fmt.Errorf("no such alias: %s", alias)
	}
	delete(aliases.data, alias)
	aliases.save()
	return nil
}

func (aliases *Aliases) load() {
	logger.Debug().Str("file", aliases.FilePath).Msg("Reading aliases")
	data, err := os.ReadFile(aliases.FilePath)
	if err != nil {
		logger.Fatal().Str("file", aliases.FilePath).Err(err).Msg("Failed to read aliases file")
	}
	contents, err := toml.LoadBytes(data)
	if err != nil {
		logger.Fatal().Str("file", aliases.FilePath).Err(err).Msg("Failed to deserialize aliases data")
	}
	for _, key := range contents.Keys() {
		value := contents.Get(key)
		clusterID, ok := value.(string)
		if ok {
			aliases.data[key] = clusterID
		} else {
			logger.Warn().Str("file", aliases.FilePath).Str("alias", key).
				Msgf("Alias value is not a string - ignored")
		}
	}
}

func (aliases *Aliases) save() {
	logger.Debug().Str("file", aliases.FilePath).Msg("Writing aliases")
	contents, err := toml.TreeFromMap(aliases.data)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to serialize aliases")
	}
	file, err := os.OpenFile(aliases.FilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModeExclusive|0644)
	if err != nil {
		logger.Fatal().Str("file", aliases.FilePath).Err(err).
			Msg("Failed to open aliases file for writing")
	}
	_, err = contents.WriteTo(file)
	if err != nil {
		logger.Fatal().Str("file", aliases.FilePath).Err(err).Msg("Failed to write aliases file")
	}
}

func ParseClusterIdentifier(aliasOrClusterID string) (string, error) {
	if aliasOrClusterID == "" {
		return "", nil
	}
	result, exists := NewAliases().Get(aliasOrClusterID)
	if !exists {
		// check for a valid cluster id if not in the aliases
		_, err := uuid.Parse(result)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}
