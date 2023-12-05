package api

import (
	"net/http"

	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/chart"
)

// TODO: could be used without bundle using the remote download option
func isChartEnabled() bool {
	return bundle.IsBundled()
}

type ChartInstallRequest struct {
	kubeConfigPath string
	config         chart.Configuration
}

func postChartInstall(w http.ResponseWriter, r *http.Request) {
	if !isChartEnabled() {
		disabledResponse(w, r)
		return
	}

	var installRequest ChartInstallRequest
	if err := parseJSONRequest(r, &installRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	kubeConfig, err := chart.ReadKubeConfig(installRequest.kubeConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = chart.InstallOrUpgrade(
		r.Context(),
		&installRequest.config,
		&chart.HelmOptions{KubeConfig: kubeConfig},
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
