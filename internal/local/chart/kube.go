package chart

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/weka/gohomecli/internal/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const KubeConfigPath = "/etc/rancher/k3s/k3s.yaml"

// ReadKubeConfig reads the kubeconfig from the given path with fallback to ~/.kube/config
func ReadKubeConfig(kubeConfigPath string) ([]byte, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
		if kubeConfigPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("unable to read kubeconfig: %w", err)
			}

			kubeConfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}

	logger.Debug().Str("kubeConfigPath", kubeConfigPath).Msg("Reading kubeconfig")
	kubeConfig, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read kubeconfig")
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return kubeConfig, nil
}

func NewHelmClient(ctx context.Context, opts *HelmOptions) (helmclient.Client, error) {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	logger.Info().
		Str("namespace", namespace).
		Str("kubeContext", opts.KubeContext).
		Msg("Configuring helm client")

	// kubeContext override isn't working - https://github.com/mittwald/go-helm-client/issues/127
	return helmclient.NewClientFromKubeConf(&helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
			DebugLog: func(format string, v ...interface{}) {
				logger.Debug().Msgf(format, v...)
			},
			Output: utils.NewWritterFunc(func(b []byte) {
				logger.Info().Msg(string(b))
			}),
		},
		KubeContext: opts.KubeContext,
		KubeConfig:  opts.KubeConfig,
	})
}

type warningEvent struct {
	Name    string
	Message string
}

// watchWarningEvents watches for warning events
func watchWarningEvents(ctx context.Context, namespace string, kubeconfig []byte) (chan warningEvent, func(), error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	watcher, err := k8s.CoreV1().Events(namespace).
		Watch(ctx, v1.ListOptions{TypeMeta: v1.TypeMeta{Kind: "Pod"}})
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan warningEvent, 10000)

	go func() {
		for evt := range watcher.ResultChan() {
			switch ev := evt.Object.(type) {
			case *corev1.Event:
				if ev.Type == "Warning" && ev.Reason != "BackOff" {
					logger.Debug().Str("name", ev.Name).Msg(ev.Message)
					ch <- warningEvent{Name: ev.Name, Message: ev.Message}
				}
			case *v1.Status:
				logger.Debug().Msg(ev.Message)
			default:
				logger.Debug().Msgf("Uknown event type: %T", ev)
			}
		}
		close(ch)
	}()

	return ch, watcher.Stop, nil
}
