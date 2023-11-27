package cli

import (
	"github.com/weka/gohomecli/internal/cli/app"

	_ "github.com/weka/gohomecli/internal/cli/api"
	_ "github.com/weka/gohomecli/internal/cli/config"
	_ "github.com/weka/gohomecli/internal/cli/k3s"
)

func Execute() {
	app.Execute()
}
