package api

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(eventsCmd)
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
		eventsCmd.Flags().IntVar(&eventsCmdArgs.Limit, "limit", 1000,
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
		eventsCmd.Flags().BoolVar(&eventsCmdArgs.Wide, "wide", false,
			"show more information on events, specifically their params")
		eventsCmd.Flags().BoolVar(&eventsCmdArgs.Json, "json", false,
			"Use JSON output format")
		// eventsCmd.Flags().StringVar(&eventsCmdArgs.Params, "param", "",
		//	"show events having these parameters")
	})
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
	Wide               bool
	Json               bool
	// Params             string
}{}

var eventsCmd = &cobra.Command{
	Use:     "events <cluster-id>",
	Aliases: []string{"events"}, // backward compatibility
	Short:   "Show cluster events",
	Long:    "Show cluster events",
	GroupID: "API",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startTime, err := utils.ParseTime(eventsCmdArgs.StartTime)
		if err != nil {
			utils.UserError(err.Error())
		}
		endTime, err := utils.ParseTime(eventsCmdArgs.EndTime)
		if err != nil {
			utils.UserError(err.Error())
			return
		}
		if eventsCmdArgs.ReverseSort {
			// Need server side support for this. Legacy CLI used to get a single
			// page of events and then reverse it, but here we want to support
			// pagination so we have to have the server do the sorting.
			utils.UserError("--reverse is not supported yet")
			return
		}
		clusterID, err := env.ParseClusterIdentifier(args[0])
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", args[0]))
		}
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
			Limit:              eventsCmdArgs.Limit,
			Wide:               eventsCmdArgs.Wide,
			// Params:             eventsCmdArgs.Params,
		})
		if err != nil {
			utils.UserError(err.Error())
			return
		}
		// query.Options.NoAutoFetchNextPage = false
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
		if eventsCmdArgs.Wide {
			headers = append(headers, "Params")
		}
		if eventsCmdArgs.Json {
			for {
				event, err := query.NextEvent()
				if err != nil {
					utils.UserError(err.Error())
				}
				if event == nil {
					break
				}
				val, err := json.MarshalIndent(event, "", "    ")
				if err != nil {
					utils.UserError(err.Error())
					return
				}
				fmt.Println(string(val))
			}
			return
		}
		numEvents := 0
		utils.RenderTableRows(headers, func() []string {
			// Limit
			if numEvents >= eventsCmdArgs.Limit {
				return nil
			}
			// Get event
			event, err := query.NextEvent()
			if err != nil {
				utils.UserError(err.Error())
			}
			if event == nil {
				return nil
			}
			numEvents++
			// Build row
			row := utils.NewTableRow(len(headers))
			row.Append(
				utils.FormatTime(event.Time),
				utils.FormatEventType(event.EventType),
				event.Category,
			)
			if eventsCmdArgs.ShowEventIDs {
				row.Append(utils.FormatUUID(event.CloudID))
			}
			if eventsCmdArgs.ShowIngestTime {
				row.Append(utils.FormatTime(event.IngestTime))
			}
			row.Append(
				utils.FormatBoolean(event.IsBackend),
				utils.FormatNodeID(event.NodeID),
				strconv.FormatInt(event.OrganizationID, 10),
				event.Permission,
				utils.FormatBoolean(event.Processed),
				utils.FormatEventSeverity(event.Severity),
			)
			if eventsCmdArgs.ShowProcessingTime {
				row.Append(strconv.FormatFloat(event.ComputeProcessingTime(), 'f', 2, 64))
			}
			if eventsCmdArgs.Wide {
				var jsonRawUnescaped json.RawMessage // json raw with unescaped unicode chars
				jsonRawUnescaped, _ = utils.UnescapeUnicodeCharactersInJSON(event.Params)
				row.Append(string(jsonRawUnescaped))
			}
			return row.Cells
		})
	},
}
