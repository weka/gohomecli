package chart

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/weka/gohomecli/internal/utils"
)

const (
	KubeConfigPath = "/etc/rancher/k3s/k3s.yaml"
	copyFromPodTimeout = 10 * time.Minute
)

// K8sClient holds a Kubernetes clientset and REST config
type K8sClient struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

// NewKubernetesClient creates a new Kubernetes client from the default kubeconfig
func NewKubernetesClient() (*K8sClient, error) {
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

	return &K8sClient{
		Clientset: clientset,
		Config:    config,
	}, nil
}

// K8sExecClient holds Kubernetes client info for executing commands in a specific pod
type K8sExecClient struct {
	K8s       *K8sClient
	PodName   string
	Container string
}

// NewK8sExecClient creates a client for executing commands in a pod found by label selector
func NewK8sExecClient(ctx context.Context, labelSelector, container string) (*K8sExecClient, error) {
	k8s, err := NewKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	pods, err := k8s.Clientset.CoreV1().Pods(ReleaseNamespace).List(ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pod found with label selector %q", labelSelector)
	}

	return &K8sExecClient{
		K8s:       k8s,
		PodName:   pods.Items[0].Name,
		Container: container,
	}, nil
}

// newExecutor creates a remotecommand executor for the given command
func (c *K8sExecClient) newExecutor(command []string) (remotecommand.Executor, error) {
	req := c.K8s.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.PodName).
		Namespace(ReleaseNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: c.Container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	return remotecommand.NewSPDYExecutor(c.K8s.Config, "POST", req.URL())
}

// Exec runs a command in the pod and returns the output
func (c *K8sExecClient) Exec(ctx context.Context, command ...string) (string, error) {
	executor, err := c.newExecutor(command)
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("exec failed: %s - %w", stderr.String(), err)
	}

	return stdout.String(), nil
}

// CopyFromPod copies a file from the pod to the local filesystem using streaming
func (c *K8sExecClient) CopyFromPod(ctx context.Context, remotePath, localPath string) error {
	// Use sh -c to cd first, then tar - more portable across tar implementations (busybox, gnu)
	tarCmd := fmt.Sprintf("cd %q && tar cf - %q", filepath.Dir(remotePath), filepath.Base(remotePath))

	executor, err := c.newExecutor([]string{"sh", "-c", tarCmd})
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Add timeout to prevent indefinite hangs on large files or slow connections
	copyCtx, cancel := context.WithTimeout(ctx, copyFromPodTimeout)
	defer cancel()

	// Use pipe for streaming (memory efficient for large files)
	reader, writer := io.Pipe()
	defer reader.Close() // Ensure cleanup even on panic
	var stderr bytes.Buffer
	execErrCh := make(chan error, 1)

	// Stream tar from pod in background
	go func() {
		defer writer.Close()
		execErrCh <- executor.StreamWithContext(copyCtx, remotecommand.StreamOptions{
			Stdout: writer,
			Stderr: &stderr,
		})
	}()

	// Extract file from tar stream
	extractErr := extractTarFile(reader, localPath)

	// Close reader to unblock the goroutine if extraction fails/completes
	reader.Close()

	// Wait for goroutine to complete and get its error
	execErr := <-execErrCh

	// Log stderr if there's any output
	if stderr.Len() > 0 {
		logger.Debug().Str("stderr", stderr.String()).Msg("CopyFromPod tar stderr")
	}

	if extractErr != nil {
		logger.Debug().Err(extractErr).Str("stderr", stderr.String()).Msg("CopyFromPod extract failed")

		return extractErr
	}
	if execErr != nil {
		// Ignore "closed pipe" error when extraction succeeded - this happens when we close
		// the reader after extracting the file but before tar finishes streaming
		if isClosedPipeError(execErr) {
			return nil
		}
		// Check for context timeout
		if errors.Is(execErr, context.DeadlineExceeded) {
			return fmt.Errorf("copy timed out after %v: %w", copyFromPodTimeout, execErr)
		}

		return fmt.Errorf("tar exec failed: %s - %w", stderr.String(), execErr)
	}

	return nil
}

// isClosedPipeError checks if the error is a "closed pipe" error which is expected
// when we close the reader early after successfully extracting the file
func isClosedPipeError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "closed pipe")
}

// extractTarFile extracts a single file from a tar stream
func extractTarFile(reader io.Reader, destPath string) error {
	tr := tar.NewReader(reader)
	entryCount := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("file not found in tar archive (found %d entries)", entryCount)
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		entryCount++

		// Accept both TypeReg ('0') and TypeRegA ('\x00') for compatibility
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close() //nolint:errcheck // error on close after successful write is acceptable

			_, copyErr := io.Copy(outFile, tr)
			if copyErr != nil {
				return fmt.Errorf("failed to copy file: %w", copyErr)
			}

			return nil
		}
	}
}

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
	k8s, err := NewKubernetesClient()
	if err != nil {
		return nil, err
	}

	pods, err := k8s.Clientset.CoreV1().Pods(ReleaseNamespace).List(ctx, v1.ListOptions{})
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

// GetPVCName returns the name of a PVC found by label selector
func GetPVCName(ctx context.Context, labelSelector string) (string, error) {
	k8s, err := NewKubernetesClient()
	if err != nil {
		return "", err
	}

	pvcs, err := k8s.Clientset.CoreV1().PersistentVolumeClaims(ReleaseNamespace).List(ctx, v1.ListOptions{
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

// GetConfigMapData returns a specific key's value from a ConfigMap found by label selector
func GetConfigMapData(ctx context.Context, labelSelector, key string) (string, error) {
	k8s, err := NewKubernetesClient()
	if err != nil {
		return "", err
	}

	configMaps, err := k8s.Clientset.CoreV1().ConfigMaps(ReleaseNamespace).List(ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", err
	}

	if len(configMaps.Items) == 0 {
		return "", fmt.Errorf(
			"no ConfigMap found with label selector %q in namespace %s",
			labelSelector,
			ReleaseNamespace,
		)
	}

	value, ok := configMaps.Items[0].Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in ConfigMap %s", key, configMaps.Items[0].Name)
	}

	return value, nil
}

// GetIngressAddress returns the address of the ingress in ReleaseNamespace namespace
func GetIngressAddress(ctx context.Context) (string, error) {
	k8s, err := NewKubernetesClient()
	if err != nil {
		return "", err
	}

	ingress, err := k8s.Clientset.NetworkingV1().
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
