package config_v1

import (
	"gopkg.in/guregu/null.v4/zero"
)

type TLSConfig struct {
	Enabled zero.Bool `json:"enabled,omitempty"` // ingress tls enabled
	Cert    string    `json:"cert,omitempty"`    // ingress tls cert
	Key     string    `json:"key,omitempty"`     // ingress tls key
}

type SMTPConfig struct {
	Host        string    `json:"host,omitempty"`        // smtp server host
	Port        int       `json:"port,omitempty"`        // smtp server port
	User        string    `json:"user,omitempty"`        // smtp server user
	Password    string    `json:"password,omitempty"`    // smtp server password
	Insecure    zero.Bool `json:"insecure,omitempty"`    // smtp insecure connection
	Sender      string    `json:"sender,omitempty"`      // smtp sender name
	SenderEmail string    `json:"senderEmail,omitempty"` // smtp sender email
}

type RetentionConfig struct {
	Diagnostics int `json:"diagnostics,omitempty"` // diagnostics retention days
	Events      int `json:"events,omitempty"`      // events retention days
}

type ForwardingConfig struct {
	Enabled                   zero.Bool `json:"enabled,omitempty"`                   // forwarding enabled
	Url                       string    `json:"url,omitempty"`                       // forwarding url override
	EnableEvents              bool      `json:"enableEvents,omitempty"`              // forwarding enable events
	EnableUsageReports        bool      `json:"enableUsageReports,omitempty"`        // forwarding enable usage reports
	EnableAnalytics           bool      `json:"enableAnalytics,omitempty"`           // forwarding enable analytics
	EnableDiagnostics         bool      `json:"enableDiagnostics,omitempty"`         // forwarding enable diagnostics
	EnableStats               bool      `json:"enableStats,omitempty"`               // forwarding enable stats
	EnableClusterRegistration bool      `json:"enableClusterRegistration,omitempty"` // forwarding enable cluster registration
}

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	Host        string   `json:"host,omitempty"`         // ingress host
	NodeIP      string   `json:"node_ip,omitempty"`      // node ip to bind on as primary internal ip
	ExternalIPs []string `json:"external_ips,omitempty"` // list of external ip addresses, optional

	TLS           TLSConfig        `json:"tls"`
	SMTP          SMTPConfig       `json:"smtp"`
	RetentionDays RetentionConfig  `json:"retentionDays"`
	Forwarding    ForwardingConfig `json:"forwarding"`

	Autoscaling     zero.Bool `json:"autoscaling,omitempty"`        // enable services autoscaling
	WekaNodesServed int       `json:"wekaNodesMonitored,omitempty"` // number of weka nodes to monitor, controls load preset
}
