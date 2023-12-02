package api

import (
	"net/http"
)

func Router() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/v1/health", allowedMethods(getHealth, http.MethodGet))
	router.HandleFunc("/v1/features", allowedMethods(getFeatures, http.MethodGet))
	router.HandleFunc("/v1/chart/install", allowedMethods(postChartInstall, http.MethodPost))
	router.HandleFunc("/v1/k3s/install", allowedMethods(postK3sInstall, http.MethodPost))
	router.HandleFunc("/v1/k3s/upgrade", allowedMethods(postK3sUpgrade, http.MethodPost))
	router.HandleFunc("/v1/k3s/import-images", allowedMethods(postImportImages, http.MethodPost))

	return router
}

type StatusResponse struct {
	Status string `json:"status"`
}

type FeaturesResponse struct {
	Chart bool `json:"chart"`
	K3s   bool `json:"k3s"`
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, r, StatusResponse{Status: "ok"})
}

func getFeatures(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, r, FeaturesResponse{Chart: isChartEnabled(), K3s: isK3sEnabled()})
}
