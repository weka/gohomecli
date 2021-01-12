package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
	"strconv"
	"time"
)

func init() {
	rootCmd.AddCommand(eventsCmd)
}

var eventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"events"}, // backward compatibility
	Short:   "Show events",
	Long:    "Show events",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := args[0]
		api := client.GetClient()
		query, err := api.QueryEvents(clusterID)
		if err != nil {
			utils.UserError(err.Error())
		}
		query.Options.NoAutoFetchNextPage = true
		headers := []string{
			"Time", "Type", "Category",
			"Is Backend", "Node", "Org ID",
			"Params", "Permission", "Processed",
			"Severity",
		}
		utils.RenderTableRows(
			headers,
			func() []string {
				event, err := query.NextEvent()
				if err != nil {
					utils.UserError(err.Error())
				}
				if event == nil {
					return nil
				}
				return []string{
					utils.Colorize(utils.ColorCyan, event.Time.Format(time.RFC3339)),
					utils.Colorize(utils.ColorBlue, event.EventType),
					event.Category,
					"N/A",  // is_backend
					event.NodeID,
					strconv.FormatInt(event.OrganizationID, 10),
					string(event.Fields),
					event.Permission,
					utils.BoolToYesNo(event.Processed),
					event.Severity,
				}
			})
	},
}
