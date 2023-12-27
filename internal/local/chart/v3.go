package chart

import (
	"errors"
	"fmt"
)

var valuesGeneratorV3 *yamlGenerator

func configureIngress(configuration *Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntryIfPtrSet(cfg, "ingress.host", configuration.Host),
		writeMapEntryIfPtrSet(cfg, "workers.alertsDispatcher.emailLinkDomainName", configuration.Host),
		writeMapEntryIfPtrSet(cfg, "ingress.tls.enabled", configuration.TLS),
		writeMapEntryIfPtrSet(cfg, "ingress.tls.cert", configuration.TLSCert),
		writeMapEntryIfPtrSet(cfg, "ingress.tls.key", configuration.TLSKey),
	)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func configureSMTP(configuration *Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntryIfPtrSet(cfg, "smtp.connection.host", configuration.SMTPHost),
		writeMapEntryIfPtrSet(cfg, "smtp.connection.port", configuration.SMTPPort),
		writeMapEntryIfPtrSet(cfg, "smtp.connection.username", configuration.SMTPUser),
		writeMapEntryIfPtrSet(cfg, "smtp.connection.password", configuration.SMTPPassword),
		writeMapEntryIfPtrSet(cfg, "smtp.connection.insecure", configuration.SMTPInsecure),
		writeMapEntryIfPtrSet(cfg, "smtp.senderEmailName", configuration.SMTPSender),
		writeMapEntryIfPtrSet(cfg, "smtp.senderEmail", configuration.SMTPSenderEmail),
	)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func configureRetention(configuration *Configuration) (yamlMap, error) {
	cfg := make(yamlMap)
	if configuration.DiagnosticsRetentionDays != nil {
		retention := fmt.Sprintf("%dd", *configuration.DiagnosticsRetentionDays)
		if err := writeMapEntry(cfg, "jobs.garbageCollector.diagnostics.maxAge", retention); err != nil {
			return nil, err
		}
	}

	if configuration.EventsRetentionDays != nil {
		retention := fmt.Sprintf("%dd", *configuration.EventsRetentionDays)
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

func configureResources(configuration *Configuration) (yamlMap, error) {
	if configuration.WekaNodesServed == nil {
		return yamlMap{}, nil
	}

	var preset *appPreset
	for i := range resourcePresets {
		if *configuration.WekaNodesServed >= resourcePresets[i].NodesThreshold {
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
	if !configuration.Autoscaling {
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

func configureForwarding(configuration *Configuration) (yamlMap, error) {
	if !configuration.ForwardingEnabled {
		return yamlMap{}, nil
	}

	cfg := make(yamlMap)
	err := errors.Join(
		writeMapEntry(cfg, "forwarding.enabled", true),
		writeMapEntryIfPtrSet(cfg, "forwarding.url", configuration.ForwardingUrl),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableEvents", configuration.ForwardingEnableEvents),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableUsageReports", configuration.ForwardingEnableUsageReports),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableAnalytics", configuration.ForwardingEnableAnalytics),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableDiagnostics", configuration.ForwardingEnableDiagnostics),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableStats", configuration.ForwardingEnableStats),
		writeMapEntryIfPtrSet(cfg, "forwarding.categories.enableClusterRegistration", configuration.ForwardingEnableClusterRegistration),
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

func generateValuesV3(configuration *Configuration) (map[string]interface{}, error) {
	return valuesGeneratorV3.Generate(configuration)
}