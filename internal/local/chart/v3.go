package chart

import (
	"errors"
	"fmt"

	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

var valuesGeneratorV3 *yamlGenerator

func configureIngress(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntryIfSet(cfg, "ingress.host", configuration.Host),
		writeMapEntryIfSet(cfg, "workers.alertsDispatcher.emailLinkDomainName", configuration.Host),
		writeMapEntryIfSet(cfg, "ingress.tls.enabled", configuration.TLS.Enabled),
		writeMapEntryIfSet(cfg, "ingress.tls.cert", configuration.TLS.Cert),
		writeMapEntryIfSet(cfg, "ingress.tls.key", configuration.TLS.Key),
	)

	if err != nil {
		return nil, err
	}

	return cfg, nil
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

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func configureRetention(configuration *config_v1.Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	if configuration.RetentionDays.Diagnostics != 0 {
		retention := fmt.Sprintf("%dd", configuration.RetentionDays.Diagnostics)
		if err := writeMapEntry(cfg, "jobs.garbageCollector.diagnostics.maxAge", retention); err != nil {
			return nil, err
		}
	}

	if configuration.RetentionDays.Events != 0 {
		retention := fmt.Sprintf("%dd", configuration.RetentionDays.Events)
		if err := writeMapEntry(cfg, "jobs.garbageCollector.events.maxAge", retention); err != nil {
			return nil, err
		}
	}

	return cfg, nil
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
	if utils.IsSetP(configuration.Autoscaling) {
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

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func configureForwarding(configuration *config_v1.Configuration) (yamlMap, error) {
	if utils.IsSetP(configuration.Forwarding.Enabled) {
		return yamlMap{}, nil
	}

	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntry(cfg, "forwarding.enabled", true),
		writeMapEntryIfSet(cfg, "forwarding.url", configuration.Forwarding.Url),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableEvents", configuration.Forwarding.EnableEvents),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableUsageReports", configuration.Forwarding.EnableUsageReports),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableAnalytics", configuration.Forwarding.EnableAnalytics),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableDiagnostics", configuration.Forwarding.EnableDiagnostics),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableStats", configuration.Forwarding.EnableStats),
		writeMapEntryIfSet(cfg, "forwarding.categories.enableClusterRegistration", configuration.Forwarding.EnableClusterRegistration),
	)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func init() {
	valuesGeneratorV3 = &yamlGenerator{
		visitors: map[string]configVisitor{},
	}

	valuesGeneratorV3.AddVisitor("ingress", configureIngress)
	valuesGeneratorV3.AddVisitor("smtp", configureSMTP)
	valuesGeneratorV3.AddVisitor("retention", configureRetention)
	valuesGeneratorV3.AddVisitor("resources", configureResources)
	valuesGeneratorV3.AddVisitor("forwarding", configureForwarding)
}

func generateValuesV3(configuration *config_v1.Configuration) (map[string]interface{}, error) {
	return valuesGeneratorV3.Generate(configuration)
}
