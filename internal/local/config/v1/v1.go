package config_v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

type TLSConfig struct {
	Cert string `json:"cert,omitempty"` // ingress tls cert
	Key  string `json:"key,omitempty"`  // ingress tls key
}

type SMTPConfig struct {
	Insecure    *bool  `json:"insecure,omitempty"`
	Host        string `json:"host,omitempty"`
	User        string `json:"user,omitempty"`
	Password    string `json:"password,omitempty"`
	Sender      string `json:"sender,omitempty"`
	SenderEmail string `json:"senderEmail,omitempty"`
	Port        int    `json:"port,omitempty"`
}

type RetentionConfig struct {
	Diagnostics int `json:"diagnostics,omitempty"` // diagnostics retention days
	Events      int `json:"events,omitempty"`      // events retention days
	Stats       int `json:"stats,omitempty"`       // stats retention days
}

type ForwardingConfig struct {
	Enabled                   *bool  `json:"enabled,omitempty"`                   // forwarding enabled
	Url                       string `json:"url,omitempty"`                       // forwarding url override
	EnableEvents              bool   `json:"enableEvents,omitempty"`              // forwarding enable events
	EnableUsageReports        bool   `json:"enableUsageReports,omitempty"`        // forwarding enable usage reports
	EnableAnalytics           bool   `json:"enableAnalytics,omitempty"`           // forwarding enable analytics
	EnableDiagnostics         bool   `json:"enableDiagnostics,omitempty"`         // forwarding enable diagnostics
	EnableStats               bool   `json:"enableStats,omitempty"`               // forwarding enable stats
	EnableClusterRegistration bool   `json:"enableClusterRegistration,omitempty"` // forwarding enable cluster registration
}

type ProxyConfig struct {
	URL     string   `json:"url,omitempty"`
	NoProxy []string `json:"noProxy,omitempty"`
}

var noProxyIPs = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"cluster.local",
	"localhost",
	"::1/128",   // IPv6 loopback
	"fc00::/7",  // Unique Local Addresses (ULA)
	"fe80::/10", // Link-Local Unicast
	"ff00::/8",  // Multicast (Optional)
}

func (p ProxyConfig) NoProxyWithDefaults() []string {
	return append(noProxyIPs, p.NoProxy...)
}

// GithubSSOConfig is a custom configuration to login with github SSO.
type GithubSSOConfig struct {
	ClientID     string `json:"clientID,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	EmailDomain  string `json:"emailDomain,omitempty"`
}

// RecordingsConfig configures the remote session recordings storage.
type RecordingsConfig struct {
	Size string `json:"size,omitempty"` // PVC size (e.g., "50Gi")
}

// RemoteSessionConfig configures the remote session client.
type RemoteSessionConfig struct {
	Recordings RecordingsConfig `json:"recordings,omitempty"`
}

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	HelmOverrides   map[string]any      `json:"helmOverrides,omitempty"`
	Autoscaling     *bool               `json:"autoscaling,omitempty"`
	GithubSSO       GithubSSOConfig     `json:"githubSSO"`
	TLS             TLSConfig           `json:"tls,omitempty"`
	IPv6            string              `json:"ip6,omitempty"`
	IPv4            string              `json:"ip,omitempty"`
	Host            string              `json:"host,omitempty"`
	RemoteSession   RemoteSessionConfig `json:"remoteSession,omitempty"`
	SMTP            SMTPConfig          `json:"smtp,omitempty"`
	Proxy           ProxyConfig         `json:"proxy,omitempty"`
	Forwarding      ForwardingConfig    `json:"forwarding,omitempty"`
	K3SArgs         []string            `json:"k3sArgs,omitempty"`
	RetentionDays   RetentionConfig     `json:"retentionDays,omitempty"`
	WekaNodesServed int                 `json:"wekaNodesMonitored,omitempty"`
}

func (c Configuration) Validate() error {
	if c.RemoteSession.Recordings.Size != "" {
		if _, err := resource.ParseQuantity(c.RemoteSession.Recordings.Size); err != nil {
			return fmt.Errorf(
				"invalid remoteSession.recordings.size %q: use Kubernetes quantity format (e.g., 10Gi, 500Mi)",
				c.RemoteSession.Recordings.Size,
			)
		}
	}

	return nil
}

func (c Configuration) LoggingSafe() Configuration {
	c.TLS.Cert = "HIDDEN"
	c.TLS.Key = "HIDDEN"
	c.SMTP.Password = "HIDDEN"
	c.GithubSSO.ClientSecret = "HIDDEN"

	return c
}
