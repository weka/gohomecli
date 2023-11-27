package cli

import (
	"github.com/weka/gohomecli/internal/cli/app"

	_ "github.com/weka/gohomecli/internal/cli/api"
	_ "github.com/weka/gohomecli/internal/cli/config"
)

func Execute() {
	app.Execute()
}
