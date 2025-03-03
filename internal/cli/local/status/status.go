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
	Short: "Get the status of pods",
	Long:  "Get the status of non-running or completed pods in home-weka-io namespace",
	RunE: func(cmd *cobra.Command, args []string) error {
		detailed, _ := cmd.Flags().GetBool("detailed")
		outputFormat, _ := cmd.Flags().GetString("output")

		pods, err := chart.GetNonRunningOrCompletedPods()
		if err != nil {
			utils.UserError(err.Error())
		}
		if len(pods) == 0 {
			utils.UserOutput("All pods are running.")
			return nil
		}
		var output []byte
		if outputFormat == "json" {
			output, err = json.Marshal(pods)
		} else {
			output, err = yaml.Marshal(pods)
		}
		if err != nil {
			utils.UserError(err.Error())
		}

		if detailed {
			utils.UserOutput(string(output))
		} else {
			utils.UserOutput("Non-running or completed pods:")
			for _, pod := range pods {
				utils.UserOutput(fmt.Sprintf("- %s", pod.Name))
			}
		}
		utils.UserError("There are non-running or completed pods.")
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolP("detailed", "d", false, "Show detailed information")
	statusCmd.Flags().StringP("output", "o", "human", "Output format (json or human)")

	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(statusCmd)
	})
}
