package status

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

func CliHook() hooks.Cli {
	var cli hooks.Cli

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get the status of Weka Home components",
		Long:  "Get the status of Weka Home components, including pods and the entire Weka Home instance",
	}

	podsCmd := &cobra.Command{
		Use:   "pods",
		Short: "Check the status of pods",
		Long:  fmt.Sprintf("Check the status of pods and get non-running ones in the %s namespace", chart.ReleaseNamespace),
		RunE:  podsRun,
	}
	podsCmd.PersistentFlags().BoolP("detailed", "d", false, "Show detailed information")
	podsCmd.PersistentFlags().StringP("output", "o", "human", "Output format (json or human)")

	statusCmd.AddCommand(podsCmd)

	wekahomeCmd := &cobra.Command{
		Use:   "wekahome",
		Short: "Check the status of the running local Weka Home instance",
		Long:  "Check the status of the running local Weka Home instance by querying the ingress address",
		RunE:  wekaHomeRun,
	}

	statusCmd.AddCommand(wekahomeCmd)

	cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(statusCmd)
	})
	return cli
}

func podsRun(cmd *cobra.Command, args []string) error {
	detailed, _ := cmd.Flags().GetBool("detailed")
	outputFormat, _ := cmd.Flags().GetString("output")

	pods, err := chart.GetNonRuninngPods(cmd.Context())
	if err != nil {
		utils.UserError(err.Error())
	}

	isHealthy := len(pods) == 0

	if outputFormat == "json" {
		return outputPodsAsJSON(pods, isHealthy, detailed)
	}
	return outputPodsAsTable(pods, isHealthy, detailed)
}

func outputPodsAsJSON(pods []chart.PodInfo, isHealthy bool, detailed bool) error {

	type StatusResponse struct {
		Healthy    bool            `json:"healthy"`
		FaultyPods []chart.PodInfo `json:"faultyPods,omitempty"`
	}

	response := StatusResponse{
		Healthy: isHealthy,
	}

	if detailed && !isHealthy {
		response.FaultyPods = pods
	}

	output, err := json.Marshal(response)
	if err != nil {
		return err
	}

	utils.UserOutputJSON(output)

	return nil
}

func outputPodsAsTable(pods []chart.PodInfo, isHealthy bool, detailed bool) error {
	if isHealthy {
		utils.UserNote("All pods are running.")
		return nil
	}
	utils.UserWarning("Some pods are not running.")
	if detailed {
		utils.UserWarning("Faulty pods:")
		headers := []string{"Pod Name", "Status", "Reason"}
		index := 0
		utils.RenderTableRows(headers, func() []string {
			if index < len(pods) {
				pod := pods[index]
				index++
				return []string{
					pod.Name,
					pod.Status,
					pod.Reason,
				}
			}
			return nil
		})
	}

	return nil
}

func wekaHomeRun(cmd *cobra.Command, args []string) error {
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
}
