package config_v1

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	Host        *string  `json:"host"`         // ingress host
	NodeIP      string   `json:"node_ip"`      // node ip to bind on as primary internal ip
	ExternalIPs []string `json:"external_ips"` // list of external ip addresses, optional

	TLS struct {
		Enabled *bool   `json:"enabled"` // ingress tls enabled
		Cert    *string `json:"tlsCert"` // ingress tls cert
		Key     *string `json:"tlsKey"`  // ingress tls key
	} `json:"tls"`

	SMTP struct {
		Host        *string `json:"host"`        // smtp server host
		Port        *int    `json:"port"`        // smtp server port
		User        *string `json:"user"`        // smtp server user
		Password    *string `json:"password"`    // smtp server password
		Insecure    *bool   `json:"insecure"`    // smtp insecure connection
		Sender      *string `json:"sender"`      // smtp sender name
		SenderEmail *string `json:"senderEmail"` // smtp sender email
	} `json:"smtp"`

	RetentionDays struct {
		Diagnostics *int `json:"diagnostics"` // diagnostics retention days
		Events      *int `json:"events"`      // events retention days
	} `json:"retention_days"`

	Forwarding struct {
		Enabled                   bool    `json:"enabled"`                   // forwarding enabled
		Url                       *string `json:"url"`                       // forwarding url override
		EnableEvents              *bool   `json:"enableEvents"`              // forwarding enable events
		EnableUsageReports        *bool   `json:"enableUsageReports"`        // forwarding enable usage reports
		EnableAnalytics           *bool   `json:"enableAnalytics"`           // forwarding enable analytics
		EnableDiagnostics         *bool   `json:"enableDiagnostics"`         // forwarding enable diagnostics
		EnableStats               *bool   `json:"enableStats"`               // forwarding enable stats
		EnableClusterRegistration *bool   `json:"enableClusterRegistration"` // forwarding enable cluster registration
	} `json:"forwarding"`

	Autoscaling     bool `json:"autoscaling"`        // enable services autoscaling
	WekaNodesServed *int `json:"wekaNodesMonitored"` // number of weka nodes to monitor, controls load preset
}
