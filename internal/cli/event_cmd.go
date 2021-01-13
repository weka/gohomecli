package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/client"
	"github.com/weka/gohomecli/internal/utils"
	"strconv"
)

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.Flags().BoolVar(&eventsCmdArgs.HideInternal, "hide-internal", false,
		"do not show internal events")
}

var eventsCmdArgs = struct {
	HideInternal bool
}{}

var eventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"events"}, // backward compatibility
	Short:   "Show events",
	Long:    "Show events",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := args[0]
		api := client.GetClient()
		query, err := api.QueryEvents(clusterID, &client.EventQueryOptions{
			WithInternalEvents: !eventsCmdArgs.HideInternal,
		})
		if err != nil {
			utils.UserError(err.Error())
		}
		query.Options.NoAutoFetchNextPage = true
		headers := []string{
			"Time", "Type", "Category",
			"Is Backend", "Node", "Org ID",
			"Permission", "Processed", "Severity",
		}
		utils.RenderTableRows(headers, func() []string {
			event, err := query.NextEvent()
			if err != nil {
				utils.UserError(err.Error())
			}
			if event == nil {
				return nil
			}
			return []string{
				FormatTime(event.Time),
				FormatEventType(event.EventType),
				event.Category,
				FormatBoolean(event.IsBackend),
				FormatNodeID(event.NodeID),
				strconv.FormatInt(event.OrganizationID, 10),
				event.Permission,
				FormatBoolean(event.Processed),
				FormatEventSeverity(event.Severity),
			}
		})
	},
}
