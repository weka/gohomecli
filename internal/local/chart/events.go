package chart

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func watchEvents(ctx context.Context, opts *HelmOptions) error {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(opts.KubeConfig)
	if err != nil {
		return err
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	watcher, err := k8s.CoreV1().Events(namespace).Watch(ctx, v1.ListOptions{TypeMeta: v1.TypeMeta{Kind: "Pod"}})
	if err != nil {
		return err
	}

	for evt := range watcher.ResultChan() {
		switch ev := evt.Object.(type) {
		case *corev1.Event:
			if ev.Type == "Warning" {
				logger.Warn().Str("name", ev.Name).Msg(ev.Message)
			}
		default:
			logger.Debug().Msgf("Uknown event type: %T", ev)
		}
	}

	return nil
}
