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
	eventsCmd.Flags().BoolVarP(&eventsCmdArgs.ReverseSort, "reverse", "r", false,
		"sort events from oldest to newest")
	eventsCmd.Flags().BoolVar(&eventsCmdArgs.ShowEventIDs, "show-event-ids", false,
		"show event UUID generated by Weka Home")
	eventsCmd.Flags().BoolVar(&eventsCmdArgs.ShowIngestTime, "show-ingest-time", false,
		"show event ingest time")
	eventsCmd.Flags().BoolVar(&eventsCmdArgs.ShowProcessingTime, "show-processing-time", false,
		"show event processing time")
	eventsCmd.Flags().BoolVar(&eventsCmdArgs.SortByIngestTime, "by-ingest-time", false,
		"sort events by ingest time")
	eventsCmd.Flags().IntVar(&eventsCmdArgs.Limit, "limit", 0,
		"show at most this many events")
	eventsCmd.Flags().StringArrayVarP(&eventsCmdArgs.IncludeTypes, "type", "T", []string{},
		"show events of these types only")
	eventsCmd.Flags().StringArrayVarP(&eventsCmdArgs.ExcludeTypes, "exclude-type", "X", nil,
		"do not show events of these types")
	eventsCmd.Flags().IntSliceVarP(&eventsCmdArgs.NodeIDs, "node-ids", "n", nil,
		"show events emitted from these nodes only")
	eventsCmd.Flags().StringVarP(&eventsCmdArgs.MinSeverity, "min-severity", "s", "",
		"show events with this severity or higher")
	eventsCmd.Flags().StringVar(&eventsCmdArgs.StartTime, "start", "",
		"show events emitted at this time or later")
	eventsCmd.Flags().StringVar(&eventsCmdArgs.EndTime, "end", "",
		"show events emitted at this time or later")
	//eventsCmd.Flags().StringVar(&eventsCmdArgs.Params, "param", "",
	//	"show events having these parameters")
}

var eventsCmdArgs = struct {
	HideInternal       bool
	ReverseSort        bool
	ShowEventIDs       bool
	ShowIngestTime     bool
	ShowProcessingTime bool
	SortByIngestTime   bool
	Limit              int
	IncludeTypes       []string
	ExcludeTypes       []string
	NodeIDs            []int
	MinSeverity        string
	StartTime          string
	EndTime            string
	//Params             string
}{}

var eventsCmd = &cobra.Command{
	Use:     "events <cluster-id>",
	Aliases: []string{"events"}, // backward compatibility
	Short:   "Show cluster events",
	Long:    "Show cluster events",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startTime, err := ParseTime(eventsCmdArgs.StartTime)
		if err != nil {
			utils.UserError(err.Error())
		}
		endTime, err := ParseTime(eventsCmdArgs.EndTime)
		if err != nil {
			utils.UserError(err.Error())
		}
		if eventsCmdArgs.ReverseSort {
			// Need server side support for this. Legacy CLI used to get a single
			// page of events and then reverse it, but here we want to support
			// pagination so we have to have the server do the sorting.
			utils.UserError("--reverse is not supported yet")
		}
		clusterID := args[0]
		api := client.GetClient()
		query, err := api.QueryEvents(clusterID, &client.EventQueryOptions{
			WithInternalEvents: !eventsCmdArgs.HideInternal,
			SortByIngestTime:   eventsCmdArgs.SortByIngestTime,
			IncludeTypes:       eventsCmdArgs.IncludeTypes,
			ExcludeTypes:       eventsCmdArgs.ExcludeTypes,
			NodeIDs:            eventsCmdArgs.NodeIDs,
			MinSeverity:        eventsCmdArgs.MinSeverity,
			StartTime:          startTime,
			EndTime:            endTime,
			//Params:             eventsCmdArgs.Params,
		})
		if err != nil {
			utils.UserError(err.Error())
		}
		query.Options.NoAutoFetchNextPage = true
		headers := []string{"Time", "Type", "Category"}
		if eventsCmdArgs.ShowEventIDs {
			headers = append(headers, "UUID")
		}
		if eventsCmdArgs.ShowIngestTime {
			headers = append(headers, "Cloud Time")
		}
		headers = append(headers,
			"Is Backend", "Node", "Org ID", "Permission", "Processed", "Severity")
		if eventsCmdArgs.ShowProcessingTime {
			headers = append(headers, "Processing Time")
		}
		numEvents := 0
		utils.RenderTableRows(headers, func() []string {
			// Limit
			if eventsCmdArgs.Limit != 0 {
				numEvents++
				if numEvents > eventsCmdArgs.Limit {
					return nil
				}
			}
			// Get event
			event, err := query.NextEvent()
			if err != nil {
				utils.UserError(err.Error())
			}
			if event == nil {
				return nil
			}
			// Build row
			row := utils.NewTableRow(len(headers))
			row.Append(
				FormatTime(event.Time),
				FormatEventType(event.EventType),
				event.Category,
			)
			if eventsCmdArgs.ShowEventIDs {
				row.Append(FormatUUID(event.CloudID))
			}
			if eventsCmdArgs.ShowIngestTime {
				row.Append(FormatTime(event.IngestTime))
			}
			row.Append(
				FormatBoolean(event.IsBackend),
				FormatNodeID(event.NodeID),
				strconv.FormatInt(event.OrganizationID, 10),
				event.Permission,
				FormatBoolean(event.Processed),
				FormatEventSeverity(event.Severity),
			)
			if eventsCmdArgs.ShowProcessingTime {
				row.Append(strconv.FormatFloat(event.ComputeProcessingTime(), 'f', 2, 64))
			}
			return row.Cells
		})
	},
}
