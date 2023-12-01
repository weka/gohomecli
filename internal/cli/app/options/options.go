package options

import (
	"github.com/spf13/cobra"
)

type Cli []func(*cobra.Command)

func (o *Cli) AddOption(opt func(*cobra.Command)) {
	*o = append(*o, opt)
}

func (o *Cli) InitCobra(cmd *cobra.Command) {
	for i := range *o {
		(*o)[i](cmd)
	}
}
