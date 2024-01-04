package api

import (
	"net/http"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
)

func isK3sEnabled() bool {
	return bundle.IsBundled()
}

func setup(w http.ResponseWriter, r *http.Request) {
	if !isK3sEnabled() {
		disabledResponse(w, r)
		return
	}

	var config config_v1.Configuration
	if err := parseJSONRequest(r, &config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := k3s.Install(r.Context(), k3s.InstallConfig{
		Configuration: &config,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = k3s.ImportBundleImages(r.Context(), true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = chart.Install(r.Context(), &chart.HelmOptions{
		Config: &config,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
