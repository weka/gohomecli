package api

import (
	"net/http"

	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/k3s"
)

func isK3sEnabled() bool {
	return bundle.IsBundled()
}

type K3sInstallRequest struct {
	Interface   string   `json:"interface"`
	Hostname    string   `json:"hostname"`
	NodeIP      string   `json:"node_ip"`
	ExternalIPs []string `json:"external_ips"`
}

func installK3s(w http.ResponseWriter, r *http.Request) {
	if !isK3sEnabled() {
		disabledResponse(w, r)
		return
	}

	var installRequest K3sInstallRequest
	if err := parseJSONRequest(r, &installRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := k3s.Install(r.Context(), k3s.InstallConfig{
		Iface:       installRequest.Interface,
		Hostname:    installRequest.Hostname,
		NodeIP:      installRequest.NodeIP,
		ExternalIPs: installRequest.ExternalIPs,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func k3sUpgrade(w http.ResponseWriter, r *http.Request) {
	if !isK3sEnabled() {
		disabledResponse(w, r)
		return
	}

	err := k3s.Upgrade(r.Context(), k3s.UpgradeConfig{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func k3sImportImages(w http.ResponseWriter, r *http.Request) {
	if !isK3sEnabled() {
		disabledResponse(w, r)
		return
	}

	err := k3s.ImportBundleImages(r.Context(), "", true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
