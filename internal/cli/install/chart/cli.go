package chart

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
)

var installCmdOpts struct {
	kubeConfigPath string
	localChart     string
	jsonConfig     string
	remoteDownload bool
	remoteVersion  string
}

var bundlePathOverride string

var chartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Install Weka Home Helm chart",
	Long:  `Install Weka Home Helm chart on already deployed Kubernetes cluster`,
	Args:  cobra.NoArgs,
	RunE:  runInstallOrUpgrade,
}

var Cli hooks.Cli

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(chartCmd)

		chartCmd.Flags().StringVarP(&installCmdOpts.kubeConfigPath, "kube-config", "k", "", "Path to kubeconfig file")
		chartCmd.Flags().StringVarP(&installCmdOpts.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
		chartCmd.Flags().StringVarP(&installCmdOpts.jsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")
		chartCmd.Flags().BoolVarP(&installCmdOpts.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
		chartCmd.Flags().StringVar(&installCmdOpts.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
		chartCmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")
		chartCmd.Flags().StringVar(&bundlePathOverride, "bundle", "", "bundle with images to load")
		chartCmd.Flags().MarkHidden("bundle")
	})
}
