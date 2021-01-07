package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/utils"
)

func init() {
	rootCmd.AddCommand(eventCmd)
}

var eventCmd = &cobra.Command{
	Use:   "event",
	Aliases: []string{"events"}, // backward compatibility
	Short: "Show events",
	Long:  "Show events",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UserError("Not implemented")
	},
}
