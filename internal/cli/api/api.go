package api

import "github.com/spf13/cobra"

var inits []func()

func init() {
	inits = append(inits, func() {
		appCmd.AddGroup(&APIGroup)
	})
}

var APIGroup = cobra.Group{
	ID:    "API",
	Title: "WekaHome API commands",
}

var appCmd *cobra.Command

func Init(cmd *cobra.Command) {
	appCmd = cmd
	for i := range inits {
		inits[i]()
	}
}
