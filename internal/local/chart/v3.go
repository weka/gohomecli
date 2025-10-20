package chart

import (
	"errors"
	"fmt"
	"strings"

	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

var valuesGeneratorV3 *yamlGenerator

func configureIngress(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntryIfSet(cfg, "ingress.host", configuration.Host),
		writeMapEntryIfSet(cfg, "workers.alertsDispatcher.emailLinkDomainName", configuration.Host),
	)

	if configuration.TLS.Cert != "" {
		err = errors.Join(err,
			writeMapEntryIfSet(cfg, "ingress.tls.enabled", true),
			writeMapEntryIfSet(cfg, "ingress.tls.cert", configuration.TLS.Cert),
			writeMapEntryIfSet(cfg, "ingress.tls.key", configuration.TLS.Key),
		)
	}

	return cfg, err
}

func configureSMTP(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntryIfSet(cfg, "smtp.connection.host", configuration.SMTP.Host),
		writeMapEntryIfSet(cfg, "smtp.connection.port", configuration.SMTP.Port),
		writeMapEntryIfSet(cfg, "smtp.connection.username", configuration.SMTP.User),
		writeMapEntryIfSet(cfg, "smtp.connection.password", configuration.SMTP.Password),
		writeMapEntryIfSet(cfg, "smtp.connection.insecure", configuration.SMTP.Insecure),
		writeMapEntryIfSet(cfg, "smtp.senderEmailName", configuration.SMTP.Sender),
		writeMapEntryIfSet(cfg, "smtp.senderEmail", configuration.SMTP.SenderEmail),
	)

	return cfg, err
}

func configureRetention(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	var err error

	if configuration.RetentionDays.Diagnostics != 0 {
		retention := fmt.Sprintf("%dd", configuration.RetentionDays.Diagnostics)
		err = errors.Join(err,
			writeMapEntry(cfg, "jobs.garbageCollector.diagnostics.maxAge", retention),
		)
	}

	if configuration.RetentionDays.Events != 0 {
		retention := fmt.Sprintf("%dd", configuration.RetentionDays.Events)
		err = errors.Join(err,
			writeMapEntry(cfg, "jobs.garbageCollector.events.maxAge", retention),
		)
	}

	if configuration.RetentionDays.Stats != 0 {
		retention := fmt.Sprintf("%dd", configuration.RetentionDays.Stats)
		err = errors.Join(err,
			writeMapEntry(cfg, "victoria.vmstorage.retentionPeriod", retention),
		)
	}

	return cfg, err
}

type replicasPreset struct {
	Replicas int // default number of replicas
	AMin     int // autoscaling min replicas
	AMax     int // autoscaling max replicas
}

type appPreset struct {
	NodesThreshold   int            // minimum number of weka nodes apply preset
	MainApi          replicasPreset // preset for main-api
	StatsApi         replicasPreset // preset for stats-api
	StatsWorker      replicasPreset // preset for stats-worker
	ForwardingWorker replicasPreset // preset for forwarding-worker
}

var resourcePresets []appPreset = []appPreset{
	{
		NodesThreshold:   1000,
		MainApi:          replicasPreset{Replicas: 3, AMin: 3, AMax: 5},
		StatsApi:         replicasPreset{Replicas: 3, AMin: 3, AMax: 5},
		StatsWorker:      replicasPreset{Replicas: 3, AMin: 3, AMax: 10},
		ForwardingWorker: replicasPreset{Replicas: 2, AMin: 2, AMax: 5},
	},
	{
		NodesThreshold:   5000,
		MainApi:          replicasPreset{Replicas: 5, AMin: 5, AMax: 8},
		StatsApi:         replicasPreset{Replicas: 5, AMin: 5, AMax: 8},
		StatsWorker:      replicasPreset{Replicas: 10, AMin: 10, AMax: 20},
		ForwardingWorker: replicasPreset{Replicas: 3, AMin: 3, AMax: 8},
	},
}

func configureResources(configuration *config_v1.Configuration) (yamlMap, error) {
	if configuration.WekaNodesServed == 0 {
		return yamlMap{}, nil
	}

	var preset *appPreset
	for i := range resourcePresets {
		if configuration.WekaNodesServed >= resourcePresets[i].NodesThreshold {
			preset = &resourcePresets[i]

			break
		}
	}

	// default preset is used if can not match
	if preset == nil {
		return yamlMap{}, nil
	}

	cfg := make(yamlMap)
	var err error
	if !utils.IsSetP(configuration.Autoscaling) {
		err = errors.Join(
			writeMapEntry(cfg, "api.main.replicas", preset.MainApi.Replicas),
			writeMapEntry(cfg, "api.stats.replicas", preset.StatsApi.Replicas),
			writeMapEntry(cfg, "workers.stats.replicas", preset.StatsWorker.Replicas),
			writeMapEntry(cfg, "workers.forwarding.replicas", preset.ForwardingWorker.Replicas),
		)
	} else {
		err = errors.Join(
			writeMapEntry(cfg, "api.main.autoscaling.enabled", true),
			writeMapEntry(cfg, "api.main.autoscaling.minReplicas", preset.MainApi.AMin),
			writeMapEntry(cfg, "api.main.autoscaling.maxReplicas", preset.MainApi.AMax),
			writeMapEntry(cfg, "api.stats.autoscaling.enabled", true),
			writeMapEntry(cfg, "api.stats.autoscaling.minReplicas", preset.StatsApi.AMin),
			writeMapEntry(cfg, "api.stats.autoscaling.maxReplicas", preset.StatsApi.AMax),
			writeMapEntry(cfg, "workers.stats.autoscaling.enabled", true),
			writeMapEntry(cfg, "workers.stats.autoscaling.minReplicas", preset.StatsWorker.AMin),
			writeMapEntry(cfg, "workers.stats.autoscaling.maxReplicas", preset.StatsWorker.AMax),
			writeMapEntry(cfg, "workers.forwarding.autoscaling.enabled", true),
			writeMapEntry(cfg, "workers.forwarding.autoscaling.minReplicas", preset.ForwardingWorker.AMin),
			writeMapEntry(cfg, "workers.forwarding.autoscaling.maxReplicas", preset.ForwardingWorker.AMax),
		)
	}

	return cfg, err
}

func configureForwarding(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)

	enabled := true // it's enabled by default

	if configuration.Forwarding.Enabled != nil {
		enabled = *configuration.Forwarding.Enabled
	}

	err := errors.Join(
		writeMapEntry(cfg, "api.forwarding.enabled", enabled),
		writeMapEntryIfSet(cfg, "api.forwarding.url", configuration.Forwarding.Url),
		writeMapEntryIfSet(cfg, "api.forwarding.categories.enableEvents", configuration.Forwarding.EnableEvents),
		writeMapEntryIfSet(
			cfg,
			"api.forwarding.categories.enableUsageReports",
			configuration.Forwarding.EnableUsageReports,
		),
		writeMapEntryIfSet(cfg, "api.forwarding.categories.enableAnalytics", configuration.Forwarding.EnableAnalytics),
		writeMapEntryIfSet(
			cfg,
			"api.forwarding.categories.enableDiagnostics",
			configuration.Forwarding.EnableDiagnostics,
		),
		writeMapEntryIfSet(cfg, "api.forwarding.categories.enableStats", configuration.Forwarding.EnableStats),
		writeMapEntryIfSet(
			cfg,
			"api.forwarding.categories.enableClusterRegistration",
			configuration.Forwarding.EnableClusterRegistration,
		),
	)

	return cfg, err
}

func configureOverrides(configuration *config_v1.Configuration) (yamlMap, error) {
	if len(configuration.HelmOverrides) == 0 {
		return yamlMap{}, nil
	}

	cfg := make(yamlMap)

	var err error
	for key, v := range configuration.HelmOverrides {
		err = errors.Join(err, writeMapEntry(cfg, key, v))
	}

	return cfg, err
}

// configureLWH setup values to LWH specific settings.
func configureLWH(*config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		// disable gateway
		writeMapEntry(cfg, "gateway.enabled", false),
		// disable autoscaling
		writeMapEntry(cfg, "api.stats.autoscaling.enabled", false),
		writeMapEntry(cfg, "workers.stats.autoscaling.enabled", false),
		writeMapEntry(cfg, "workers.forwarding.autoscaling.enabled", false),
		// nats stream configuration
		writeMapEntry(cfg, "storage.nats.streams.events.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.events.maxBytes", 1073741824),
		writeMapEntry(cfg, "storage.nats.streams.stats.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.stats.maxBytes", 3221225472),
		writeMapEntry(cfg, "storage.nats.streams.integrations.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.integrations.maxBytes", 1048576),
		writeMapEntry(cfg, "storage.nats.streams.alerts.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.notifications.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.forwardingLow.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.forwardingLow.maxBytes", 3221225472),
		writeMapEntry(cfg, "storage.nats.streams.forwardingHigh.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.virtualStats.replicas", 1),
		writeMapEntry(cfg, "storage.nats.streams.virtualStats.maxBytes", 1073741824),
		// storage stats
		writeMapEntry(cfg, "storage.stats.useInternal", true),
		writeMapEntry(cfg, "storage.stats.useOperator", false),
		// eventsDB configuration
		writeMapEntry(cfg, "eventsdb.primary.persistence.size", "20Gi"),
		// nats configuration
		writeMapEntry(cfg, "nats.config.cluster.enabled", false),
		writeMapEntry(cfg, "nats.config.jetstream.fileStore.pvc.size", "10Gi"),
		writeMapEntry(cfg, "nats.container.patch", []any{}),
		// victoria metrics
		writeMapEntry(cfg, "victoria-metrics-k8s-stack.enabled", true),
		writeMapEntry(cfg, "prometheus-node-exporter.enabled", true),
		// license synchronizer job
		writeMapEntry(cfg, "jobs.licenseSynchronizer.enabled", true),
		writeMapEntry(cfg, "redis-cluster.cluster", yamlMap{
			"nodes":    3,
			"replicas": 0,
			"update": yamlMap{
				"currentNumberOfNodes":    3,
				"currentNumberOfReplicas": 0,
			},
		}),
		writeMapEntry(cfg, "alertmanager.config", yamlMap{
			"templates": []string{"/etc/vm/configs/**/*.tmpl"},
			"route": yamlMap{
				"receiver": "blackhole",
			},
			"receivers": []yamlMap{
				{
					"name": "blackhole",
				},
			},
		}),
		writeMapEntry(cfg, "victoria-metrics-k8s-stack.vmagent", yamlMap{
			"configReloaderExtraArgs": yamlMap{
				"enableTCP6": true,
			},
		}),
		writeMapEntry(cfg, "victoria-metrics-k8s-stack.alertmanager.enabled", false),
		writeMapEntry(cfg, "victoria-metrics-k8s-stack.vmalert.enabled", false),
		// writeMapEntry(cfg, "vmalert.enabled", false),
		// writeMapEntry(cfg, "victoria-metrics-k8s-stack.vmcluster.enabled", false),
	)

	return cfg, err
}

func configureCore(configuration *config_v1.Configuration) (yamlMap, error) {
	var (
		err error
		cfg = make(yamlMap)
	)

	if configuration.Proxy.URL != "" {
		err = errors.Join(
			writeMapEntryIfSet(cfg, "core.proxy.url", configuration.Proxy.URL),
			writeMapEntryIfSet(cfg, "core.proxy.noProxy", strings.Join(configuration.Proxy.NoProxyWithDefaults(), ",")),
		)
	}
	// github SSO configuration.
	if len(configuration.GithubSSO.ClientID) > 0 && len(configuration.GithubSSO.ClientSecret) > 0 {
		err = errors.Join(
			writeMapEntryIfSet(cfg, "core.githubSSO.enabled", true),
			writeMapEntryIfSet(cfg, "core.githubSSO.clientID", configuration.GithubSSO.ClientID),
			writeMapEntryIfSet(cfg, "core.githubSSO.clientSecret", configuration.GithubSSO.ClientSecret),
			writeMapEntryIfSet(cfg, "core.githubSSO.emailDomain", configuration.GithubSSO.EmailDomain),
		)
	}

	return cfg, err
}

func init() {
	valuesGeneratorV3 = &yamlGenerator{
		visitors: map[string]configVisitor{},
	}

	valuesGeneratorV3.MustAddVisitor("core", configureCore)
	valuesGeneratorV3.MustAddVisitor("ingress", configureIngress)
	valuesGeneratorV3.MustAddVisitor("smtp", configureSMTP)
	valuesGeneratorV3.MustAddVisitor("retention", configureRetention)
	valuesGeneratorV3.MustAddVisitor("resources", configureResources)
	valuesGeneratorV3.MustAddVisitor("forwarding", configureForwarding)
	valuesGeneratorV3.MustAddVisitor("lwh", configureLWH)
	valuesGeneratorV3.MustAddVisitor("overrides", configureOverrides)
}

func generateValuesV3(configuration *config_v1.Configuration) (map[string]any, error) {
	return valuesGeneratorV3.Generate(configuration)
}
