package api

import (
	"github.com/weka/gohomecli/internal/install/bundle"
	k3s2 "github.com/weka/gohomecli/internal/install/k3s"
	"net/http"
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

	err := k3s2.Install(r.Context(), k3s2.InstallConfig{
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

	err := k3s2.Upgrade(r.Context(), k3s2.UpgradeConfig{})
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

	err := k3s2.ImportBundleImages(r.Context(), "", true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
