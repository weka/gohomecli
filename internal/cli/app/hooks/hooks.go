package hooks

import (
	"github.com/spf13/cobra"
)

type Cli []func(*cobra.Command)

func (o *Cli) AddHook(opt func(*cobra.Command)) {
	*o = append(*o, opt)
}

func (o *Cli) InitCobra(cmd *cobra.Command) {
	for _, hook := range *o {
		hook(cmd)
	}
}
