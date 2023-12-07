package chart

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var installCmdOpts struct {
	kubeConfigPath string
	localChart     string
	jsonConfig     string
	remoteDownload bool
	remoteVersion  string
}

var (
	chartGroup = &cobra.Group{ID: "chart", Title: "Helm chart management"}

	chartCmd = &cobra.Command{
		Use:     "chart",
		Short:   "Manage Weka Home Helm chart",
		Long:    `Manage Weka Home Helm chart`,
		GroupID: "chart",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if !env.VerboseLogging {
				utils.SetGlobalLoggingLevel(utils.InfoLevel)
			}
		},
	}

	chartInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install Weka Home Helm chart",
		Long:  `Install Weka Home Helm chart on already deployed Kubernetes cluster`,
		Args:  cobra.NoArgs,
		RunE:  runInstallOrUpgrade,
	}
)

var Cli hooks.Cli

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddGroup(chartGroup)
		appCmd.AddCommand(chartCmd)

		chartInstallCmd.Flags().StringVarP(&installCmdOpts.kubeConfigPath, "kube-config", "k", "", "Path to kubeconfig file")
		chartInstallCmd.Flags().StringVarP(&installCmdOpts.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
		chartInstallCmd.Flags().StringVarP(&installCmdOpts.jsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")
		chartInstallCmd.Flags().BoolVarP(&installCmdOpts.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
		chartInstallCmd.Flags().StringVar(&installCmdOpts.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
		chartInstallCmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")

		chartCmd.AddCommand(chartInstallCmd)
	})
}
