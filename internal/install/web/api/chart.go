package api

import (
	"github.com/weka/gohomecli/internal/install/bundle"
	chart2 "github.com/weka/gohomecli/internal/install/chart"
	"net/http"
)

// TODO: could be used without bundle using the remote download option
func isChartEnabled() bool {
	return bundle.IsBundled()
}

type ChartInstallRequest struct {
	kubeConfigPath string
	config         chart2.Configuration
}

func installChart(w http.ResponseWriter, r *http.Request) {
	if !isChartEnabled() {
		disabledResponse(w, r)
		return
	}

	var installRequest ChartInstallRequest
	if err := parseJSONRequest(r, &installRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	kubeConfig, err := chart2.ReadKubeConfig(installRequest.kubeConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = chart2.InstallOrUpgrade(
		r.Context(),
		&installRequest.config,
		&chart2.HelmOptions{KubeConfig: kubeConfig},
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
