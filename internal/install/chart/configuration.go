package chart

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	Host    *string `json:"host"`    // ingress host
	TLS     *bool   `json:"tls"`     // ingress tls enabled
	TLSCert *string `json:"tlsCert"` // ingress tls cert
	TLSKey  *string `json:"tlsKey"`  // ingress tls key

	SMTPHost        *string `json:"smtpHost"`        // smtp server host
	SMTPPort        *int    `json:"smtpPort"`        // smtp server port
	SMTPUser        *string `json:"smtpUser"`        // smtp server user
	SMTPPassword    *string `json:"smtpPassword"`    // smtp server password
	SMTPInsecure    *bool   `json:"smtpInsecure"`    // smtp insecure connection
	SMTPSender      *string `json:"smtpSender"`      // smtp sender name
	SMTPSenderEmail *string `json:"smtpSenderEmail"` // smtp sender email

	DiagnosticsRetentionDays *int `json:"diagnosticsRetentionDays"` // diagnostics retention days
	EventsRetentionDays      *int `json:"eventsRetentionDays"`      // events retention days

	Autoscaling     bool `json:"autoscaling"`        // enable services autoscaling
	WekaNodesServed *int `json:"wekaNodesMonitored"` // number of weka nodes to monitor, controls load preset
}
