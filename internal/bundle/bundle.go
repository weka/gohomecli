package bundle

import (
	"github.com/rs/zerolog"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("bundle")

func SetLogger(log zerolog.Logger) {
	logger = log.With().Str("component", "bundle").Logger()
}
