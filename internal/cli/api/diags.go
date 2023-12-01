package api

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
)

var diagsDownloadBacthCmdArgs = struct {
	topic string
}{}

var diagsListCmdArgs = struct {
	topic   string
	topicId string
	Limit   int
}{}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(diagsCmd)
		diagsCmd.AddCommand(diagsListCmd)
		diagsCmd.AddCommand(diagsDownloadCmd)
		diagsCmd.AddCommand(diagsDownloadBacthCmd)
		diagsDownloadBacthCmd.Flags().StringVar(&diagsDownloadBacthCmdArgs.topic, "topic", "diags",
			"topic identifier")
		diagsListCmd.Flags().StringVar(&diagsListCmdArgs.topicId, "topic-id", "",
			"filter topic id")
		diagsListCmd.Flags().StringVar(&diagsListCmdArgs.topic, "topic", "",
			"filter topic")
		diagsListCmd.Flags().IntVar(&diagsListCmdArgs.Limit, "limit", 500,
			"show at most this many files")
	})
}

var diagsCmd = &cobra.Command{
	Use:     "diags [OPTIONS] <cluster-id>",
	Short:   "Download cluster diagnostics",
	Long:    "Download cluster diagnostics",
	GroupID: "API",
}

var diagsListCmd = &cobra.Command{
	Use:   "list <cluster-id>",
	Short: "List cluster diagnostics",
	Long:  "List cluster diagnostics",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID, err := env.ParseClusterIdentifier(args[0])
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", args[0]))
		}
		api := client.GetClient()
		options := &client.RequestOptions{}
		options.PageSize = diagsListCmdArgs.Limit
		options.Params = client.GetDiagsParams(diagsListCmdArgs.topic, diagsListCmdArgs.topicId)
		query, err := api.QueryDiags(clusterID, options)
		if err != nil {
			utils.UserError(err.Error())
		}
		headers := []string{"Upload Time", "Filename", "Hostname", "Id", "Diags Collection Id"}
		index := 0
		utils.RenderTableRows(headers, func() []string {
			if index >= diagsListCmdArgs.Limit {
				return nil
			}
			// Get event
			diag, err := query.NextDiag()
			if err != nil {
				utils.UserError(err.Error())
			}
			if diag == nil {
				return nil
			}
			// Build row
			index++
			row := utils.NewTableRow(len(headers))
			row.Append(
				utils.FormatTime(diag.UploadTime),
				diag.FileName,
				diag.HostName,
				strconv.Itoa(diag.ID),
				diag.TopicId,
			)
			return row.Cells
		})
	},
}

var diagsDownloadCmd = &cobra.Command{
	Use:   "download <cluster-id> <filename>",
	Short: "Download cluster diagnostics file",
	Long:  "Download cluster diagnostics file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID, err := env.ParseClusterIdentifier(args[0])
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", args[0]))
		}
		api := client.GetClient()
		err = api.DownloadDiags(clusterID, args[1])
		if err != nil {
			utils.UserError(err.Error())
		}
	},
}

var diagsDownloadBacthCmd = &cobra.Command{
	Use:   "download-batch <cluster-id> <topic-id>",
	Short: "Download batch diagnostic files",
	Long:  "Download batch diagnostic files",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID, err := env.ParseClusterIdentifier(args[0])
		if err != nil {
			utils.UserError(fmt.Sprintf("%s isn't a valid guid", args[0]))
		}
		api := client.GetClient()
		options := &client.RequestOptions{}
		options.Params = client.GetDiagsParams(diagsDownloadBacthCmdArgs.topic, args[1])
		query, err := api.QueryDiags(clusterID, options)
		if err != nil {
			utils.UserError(err.Error())
		}
		files := []string{}
		for {
			diag, err := query.NextDiag()
			if err != nil {
				utils.UserError(err.Error())
			}
			if diag == nil {
				break
			}
			files = append(files, diag.FileName)
		}
		if len(files) > 0 {
			err := api.DownloadManyDiags(clusterID, files)
			if err != nil {
				utils.UserError(err.Error())
			}
		} else {
			utils.UserOutput("No files found for topic:%s  topic-id: %s",
				diagsDownloadBacthCmdArgs.topic, args[1])
		}
	},
}
