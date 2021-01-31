package cli

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
	"github.com/weka/gohomecli/pkg/client"
	"strconv"
)

func init() {
	rootCmd.AddCommand(diagsCmd)
	diagsCmd.AddCommand(diagsListCmd)
	diagsCmd.AddCommand(diagsDownloadCmd)
	diagsCmd.AddCommand(diagsDownloadBacthCmd)
}

var diagsCmd = &cobra.Command{
	Use:     "diags [OPTIONS] <cluster-id>",
	Short:   "Download cluster diagnostics",
	Long:    "Download cluster diagnostics",
}

var diagsListCmd = &cobra.Command{
	Use:     "list <cluster-id>",
	Short:   "List cluster diagnostics",
	Long:    "List cluster diagnostics",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := env.ParseClusterIdentifier(args[0])
		api := client.GetClient()
		query, err := api.QueryDiags(clusterID)
		if err != nil {
			utils.UserError(err.Error())
		}
		headers := []string{"Upload Time", "Filename", "Hostname", "Id", "Diags Collection Id"}
		utils.RenderTableRows(headers, func() []string {

			// Get event
			diag, err := query.NextDiag()
			if err != nil {
				utils.UserError(err.Error())
			}
			if diag == nil {
				return nil
			}
			// Build row
			row := utils.NewTableRow(len(headers))
			row.Append(
				FormatTime(diag.UploadTime),
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
	Use:     "download <cluster-id> <filename>",
	Short:   "Download cluster diagnostics file",
	Long:    "Download cluster diagnostics file",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := env.ParseClusterIdentifier(args[0])
		api := client.GetClient()
		err := api.DownloadDiags(clusterID, args[1])
		if err != nil {
			utils.UserError(err.Error())
		}

	},
}

var diagsDownloadBacthCmd = &cobra.Command{
	Use:     "download-batch <cluster-id> <topic-id>",
	Short:   "Download batch diagnostic files",
	Long:    "Download batch diagnostic files",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := env.ParseClusterIdentifier(args[0])
		api := client.GetClient()
		query, err := api.QueryDiags(clusterID)
		if err != nil {
			utils.UserError(err.Error())
		}
		files := []string {}
		for {
			diag, err := query.NextDiag()
			if err != nil{
				utils.UserError(err.Error())
			}
			if diag == nil {
				break
			}
			if diag.TopicId == args[1]{
				files = append(files, diag.FileName)
			}
		}
		if len(files) > 0 {
			api.DownloadManyDiags(clusterID, files)
		} else {
			utils.UserOutput("No files found for topic %s", args[1])
		}
		return
	},
}