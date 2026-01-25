package chart

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	helmclient "github.com/mittwald/go-helm-client"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/weka/gohomecli/internal/utils"
)

const (
	KubeConfigPath          = "/etc/rancher/k3s/k3s.yaml"
	clusterServiceURLFormat = "http://%s.%s.svc.cluster.local:%d"
)

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
			DebugLog: func(format string, v ...any) {
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

func isNonRunningOrIncomplete(containerStatus corev1.ContainerStatus) bool {
	containerState := containerStatus.State

	return containerState.Waiting != nil ||
		(containerState.Terminated != nil && containerState.Terminated.Reason != "Completed")
}

// PodInfo represents short information about a pod
type PodInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// GetNonRuninngPods returns a list of non-running pods in the ReleaseNamespace namespace
func GetNonRuninngPods(ctx context.Context) ([]PodInfo, error) {
	kubeconfig, err := ReadKubeConfig(KubeConfigPath)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods(ReleaseNamespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var nonRunningOrCompletedPods []PodInfo
	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if isNonRunningOrIncomplete(containerStatus) {
				reason := getContainerStatusReason(containerStatus, pod.Status.Reason)
				nonRunningOrCompletedPods = append(nonRunningOrCompletedPods, PodInfo{
					Name:   pod.Name,
					Status: string(pod.Status.Phase),
					Reason: reason,
				})

				break
			}
		}
	}

	return nonRunningOrCompletedPods, nil
}

// getContainerStatusReason extracts the most specific status reason from a container
func getContainerStatusReason(containerStatus corev1.ContainerStatus, podReason string) string {
	// Try to get the waiting reason first (most common for problems like ImagePullBackOff)
	if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason != "" {
		return containerStatus.State.Waiting.Reason
	}

	// If not waiting, try terminated state (but not if it's completed normally)
	if containerStatus.State.Terminated != nil &&
		containerStatus.State.Terminated.Reason != "Completed" &&
		containerStatus.State.Terminated.Reason != "" {
		return containerStatus.State.Terminated.Reason
	}

	// If container state doesn't have a reason, fall back to pod reason
	if podReason != "" {
		return podReason
	}

	// Default for when container is not running but no specific reason is found
	return "Unknown"
}

// GetServiceURL returns the cluster-internal URL of a service found by label selector
func GetServiceURL(ctx context.Context, labelSelector string) (string, error) {
	kubeconfig, err := ReadKubeConfig(KubeConfigPath)
	if err != nil {
		return "", err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	services, err := clientset.CoreV1().Services(ReleaseNamespace).List(ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", err
	}

	if len(services.Items) == 0 {
		return "", fmt.Errorf("no service found with label selector %q in namespace %s", labelSelector, ReleaseNamespace)
	}

	svc := services.Items[0]
	if len(svc.Spec.Ports) == 0 {
		return "", fmt.Errorf("service %s has no ports defined", svc.Name)
	}

	return fmt.Sprintf(clusterServiceURLFormat,
		svc.Name, svc.Namespace, svc.Spec.Ports[0].Port), nil
}

// GetPVCName returns the name of a PVC found by label selector
func GetPVCName(ctx context.Context, labelSelector string) (string, error) {
	kubeconfig, err := ReadKubeConfig(KubeConfigPath)
	if err != nil {
		return "", err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(ReleaseNamespace).List(ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", err
	}

	if len(pvcs.Items) == 0 {
		return "", fmt.Errorf("no PVC found with label selector %q in namespace %s", labelSelector, ReleaseNamespace)
	}

	return pvcs.Items[0].Name, nil
}

// GetIngressAddress returns the address of the ingress in ReleaseNamespace namespace
func GetIngressAddress(ctx context.Context) (string, error) {
	kubeconfig, err := ReadKubeConfig(KubeConfigPath)
	if err != nil {
		return "", err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	ingress, err := clientset.NetworkingV1().
		Ingresses(ReleaseNamespace).
		Get(ctx, "wekahome", v1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" && rule.Host != "*" {
			return rule.Host, nil
		}
	}

	// If no specific host is found, return the address
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		if ip := net.ParseIP(ingress.Status.LoadBalancer.Ingress[0].IP); ip.To4() == nil { // is IPv6
			return "[" + ingress.Status.LoadBalancer.Ingress[0].IP + "]", nil
		}

		return ingress.Status.LoadBalancer.Ingress[0].IP, nil
	}

	return "", fmt.Errorf(
		"no valid host or address found for ingress %s in namespace %s",
		ingress.Name,
		ReleaseNamespace,
	)
}
