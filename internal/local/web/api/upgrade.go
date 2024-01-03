package api

import (
	"net/http"

	"github.com/weka/gohomecli/internal/local/chart"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
)

func upgrade(w http.ResponseWriter, r *http.Request) {
	if !isK3sEnabled() {
		disabledResponse(w, r)
		return
	}

	var config config_v1.Configuration
	if err := parseJSONRequest(r, &config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := k3s.Upgrade(r.Context(), k3s.UpgradeConfig{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = k3s.ImportBundleImages(r.Context(), true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = chart.Upgrade(r.Context(), &chart.HelmOptions{
		Config: &config,
	}, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
