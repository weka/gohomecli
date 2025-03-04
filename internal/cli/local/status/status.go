package status

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli hooks.Cli

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of Weka Home components",
	Long:  "Get the status of Weka Home components, including pods and the entire Weka Home instance",
}

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Check the status of pods",
	Long:  fmt.Sprintf("Check the status of pods and get non-running ones in the %s namespace", chart.ReleaseNamespace),
	RunE: func(cmd *cobra.Command, args []string) error {
		detailed, _ := cmd.Flags().GetBool("detailed")
		outputFormat, _ := cmd.Flags().GetString("output")

		if !detailed && cmd.Flags().Changed("output") {
			utils.UserError("The --output flag can only be used with the --detailed flag.")
		}

		pods, err := chart.GetNonRuninngPods()
		if err != nil {
			utils.UserError(err.Error())
		}
		if len(pods) == 0 {
			utils.UserNote("All pods are running.")
			return nil
		}
		if detailed {
			var output []byte
			if outputFormat == "json" {
				output, err = json.Marshal(pods)
				if err != nil {
					utils.UserError(err.Error())
				}
				utils.UserOutputJSON(output)
			} else {
				output, err = yaml.Marshal(pods)
				if err != nil {
					utils.UserError(err.Error())
				}
				utils.UserOutput(string(output))
			}
		} else {
			utils.UserOutput("Non-running pods:")
			for _, pod := range pods {
				utils.UserOutput(fmt.Sprintf("- Pod name: \"%s\", Pod Status: \"%s\"", pod.Name, pod.Status.Phase))
			}
		}
		utils.UserError("There are non-running pods.")
		return nil
	},
}

var wekahomeCmd = &cobra.Command{
	Use:   "wekahome",
	Short: "Check the status of the running local Weka Home instance",
	Long:  "Check the status of the running local Weka Home instance by querying the ingress address",
	RunE: func(cmd *cobra.Command, args []string) error {
		address, err := chart.GetIngressAddress()
		if err != nil {
			utils.UserError(err.Error())
		}
		if address == "" {
			errStr := fmt.Sprintf("No ingress found in the %s namespace", chart.ReleaseNamespace)
			utils.UserError(errStr)
		}

		url := fmt.Sprintf("http://%s", address)
		statusCode, err := utils.GetURLStatusCode(url)
		if err != nil {
			utils.UserError(err.Error())
		}
		if statusCode != 200 {
			errStr := fmt.Sprintf("Something wrong with WekaHome. Address: %s, HTTP Status code: %d", url, statusCode)
			utils.UserError(errStr)
		}
		utils.UserNote("WekaHome is running.")
		return nil
	},
}

func init() {
	podsCmd.PersistentFlags().BoolP("detailed", "d", false, "Show detailed information")
	podsCmd.PersistentFlags().StringP("output", "o", "human", "Output format (json or human)")

	statusCmd.AddCommand(podsCmd)
	statusCmd.AddCommand(wekahomeCmd)

	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(statusCmd)
	})
}
