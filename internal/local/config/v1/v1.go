package config_v1

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	Host        string   `json:"host,omitempty"`         // ingress host
	NodeIP      string   `json:"node_ip,omitempty"`      // node ip to bind on as primary internal ip
	ExternalIPs []string `json:"external_ips,omitempty"` // list of external ip addresses, optional

	TLS struct {
		Enabled bool   `json:"enabled"`        // ingress tls enabled
		Cert    string `json:"cert,omitempty"` // ingress tls cert
		Key     string `json:"key,omitempty"`  // ingress tls key
	} `json:"tls"`

	SMTP struct {
		Host        string `json:"host,omitempty"`        // smtp server host
		Port        int    `json:"port,omitempty"`        // smtp server port
		User        string `json:"user,omitempty"`        // smtp server user
		Password    string `json:"password,omitempty"`    // smtp server password
		Insecure    bool   `json:"insecure,omitempty"`    // smtp insecure connection
		Sender      string `json:"sender,omitempty"`      // smtp sender name
		SenderEmail string `json:"senderEmail,omitempty"` // smtp sender email
	} `json:"smtp"`

	RetentionDays struct {
		Diagnostics int `json:"diagnostics,omitempty"` // diagnostics retention days
		Events      int `json:"events,omitempty"`      // events retention days
	} `json:"retentionDays"`

	Forwarding struct {
		Enabled                   bool   `json:"enabled"`                             // forwarding enabled
		Url                       string `json:"url,omitempty"`                       // forwarding url override
		EnableEvents              bool   `json:"enableEvents,omitempty"`              // forwarding enable events
		EnableUsageReports        bool   `json:"enableUsageReports,omitempty"`        // forwarding enable usage reports
		EnableAnalytics           bool   `json:"enableAnalytics,omitempty"`           // forwarding enable analytics
		EnableDiagnostics         bool   `json:"enableDiagnostics,omitempty"`         // forwarding enable diagnostics
		EnableStats               bool   `json:"enableStats,omitempty"`               // forwarding enable stats
		EnableClusterRegistration bool   `json:"enableClusterRegistration,omitempty"` // forwarding enable cluster registration
	} `json:"forwarding"`

	Autoscaling     bool `json:"autoscaling"`                  // enable services autoscaling
	WekaNodesServed int  `json:"wekaNodesMonitored,omitempty"` // number of weka nodes to monitor, controls load preset
}
