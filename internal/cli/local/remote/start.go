package remote

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

const (
	remoteAccessLabel        = "app"
	remoteAccessValue        = "remote-access"
	remoteAccessImage        = "public.ecr.aws/weka/weka-remote-access:v1.0.0"
	sessionRecordingsPVCLabel = "app=remote-access-recordings"
)

type startOptions struct {
	clusterID     string
	clusterName   string
	sshKeysPath   string
	cloudURL      string
	hostName      string
	terminalCols  int
	terminalLines int
	debug         bool
	// Tmate server flags (optional overrides - tmate.py has built-in config for known cloud URLs)
	tmateServerHost    string
	tmateServerPort    string
	tmateServerRSA     string
	tmateServerEd25519 string
	tmateServerECDSA   string
}

func newStartCmd() *cobra.Command {
	opts := &startOptions{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new remote-access tmate session",
		Long: `Start a remote-access tmate session for remote cluster connection.
			   Tmate server config is resolved from cloud URL. Use --tmate-server-* flags for custom servers.

			   Examples:
  			   homecli local remote start --cluster-id "550e8400-..." --cluster-name "prod" --ssh-keys-path "/root/.ssh" # Start a session with cloud URL from config
			`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startRun(cmd, opts)
		},
	}

	// Required flags
	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Cluster GUID (required)")
	cmd.Flags().StringVar(&opts.clusterName, "cluster-name", "", "Human-readable cluster name (required)")
	cmd.Flags().StringVar(&opts.sshKeysPath, "ssh-keys-path", "", "Host path to SSH keys directory (required)")
	_ = cmd.MarkFlagRequired("cluster-id")
	_ = cmd.MarkFlagRequired("cluster-name")
	_ = cmd.MarkFlagRequired("ssh-keys-path")

	// Optional flags
	cmd.Flags().StringVar(&opts.cloudURL, "cloud-url", "", "Cloud Weka Home URL (default: from config)")
	cmd.Flags().StringVar(&opts.hostName, "host-name", "", "Override hostname (default: system hostname)")
	cmd.Flags().IntVar(&opts.terminalCols, "terminal-cols", 0, "Terminal width (default: 158)")
	cmd.Flags().IntVar(&opts.terminalLines, "terminal-lines", 0, "Terminal height (default: 35)")
	cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug logging")

	// Tmate server override flags (optional - tmate.py has built-in config for known cloud URLs)
	cmd.Flags().StringVar(&opts.tmateServerHost, "tmate-server-host", "", "Override tmate SSH server hostname")
	cmd.Flags().StringVar(&opts.tmateServerPort, "tmate-server-port", "", "Override tmate SSH server port")
	cmd.Flags().StringVar(&opts.tmateServerRSA, "tmate-server-rsa-fingerprint", "", "Override RSA fingerprint")
	cmd.Flags().StringVar(&opts.tmateServerEd25519, "tmate-server-ed25519-fingerprint", "", "Override Ed25519 fingerprint")
	cmd.Flags().StringVar(&opts.tmateServerECDSA, "tmate-server-ecdsa-fingerprint", "", "Override ECDSA fingerprint")

	return cmd
}

func startRun(cmd *cobra.Command, opts *startOptions) error {
	ctx := cmd.Context()

	// Resolve cloud URL from flag or config
	cloudURL := opts.cloudURL
	if cloudURL == "" && env.CurrentSiteConfig != nil {
		cloudURL = env.CurrentSiteConfig.CloudURL
	}
	if cloudURL == "" {
		return fmt.Errorf("cloud URL not found in site config, please provide --cloud-url flag explicitly")
	}

	// Get local LWH address from ingress
	lwhAddress, err := chart.GetIngressAddress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get LWH address: %w", err)
	}
	localLWHURL := "http://" + lwhAddress

	// Build webhook base URLs (tmate.py builds full paths using WEBHOOK_URL_TEMPLATE)
	webhookURLs := buildWebhookBaseURLs(cloudURL, localLWHURL)

	// Generate session ID
	sessionID := generateShortID()

	// Create Kubernetes client
	clientset, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Get recordings PVC name
	recordingsPVCName, err := chart.GetPVCName(ctx, sessionRecordingsPVCLabel)
	if err != nil {
		return fmt.Errorf("failed to find recordings PVC: %w", err)
	}

	// Create the pod
	pod := buildSessionPod(sessionID, webhookURLs, cloudURL, recordingsPVCName, opts)

	logger.Info().
		Str("sessionID", sessionID).
		Str("clusterID", opts.clusterID).
		Str("clusterName", opts.clusterName).
		Msg("Creating remote session pod...")

	createdPod, err := clientset.CoreV1().Pods(chart.ReleaseNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create session pod: %w", err)
	}

	utils.UserOutput("Remote session started successfully\n")
	utils.UserOutput("  Session ID: %s\n", sessionID)

	logger.Info().Str("pod", createdPod.Name).Msg("Remote session pod created")

	return nil
}

func buildWebhookBaseURLs(cloudURL, localAPIURL string) string {
	var urls []string

	// Add cloud base URL if configured
	if cloudURL != "" {
		urls = append(urls, strings.TrimSuffix(cloudURL, "/"))
	}

	// Always add local LWH base URL
	urls = append(urls, localAPIURL)

	return strings.Join(urls, ",")
}

func generateShortID() string {
	bytes := make([]byte, 3) // 3 bytes = 6 hex chars
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getKubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfig, err := chart.ReadKubeConfig(chart.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func buildSessionPod(sessionID, webhookURLs, cloudURL, recordingsPVCName string, opts *startOptions) *corev1.Pod {
	const sharedSocketPath = "/shared/tmate.socket"

	// Environment variables for tmate container
	tmateEnv := []corev1.EnvVar{
		{Name: "CLUSTER_ID", Value: opts.clusterID},
		{Name: "CLUSTER_NAME", Value: opts.clusterName},
		{Name: "SHARED_SOCKET", Value: sharedSocketPath},
		{Name: "CLOUD_URL", Value: cloudURL},
		{Name: "WEBHOOK_URLS", Value: webhookURLs},
	}

	// Add HOST_NAME only if explicitly provided (tmate.py uses socket.gethostname() as default)
	if opts.hostName != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "HOST_NAME", Value: opts.hostName})
	}

	// Add terminal dimensions only if explicitly provided (tmate.py defaults to 158x35)
	if opts.terminalCols > 0 {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TERMINAL_COLS", Value: fmt.Sprintf("%d", opts.terminalCols)})
	}
	if opts.terminalLines > 0 {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TERMINAL_LINES", Value: fmt.Sprintf("%d", opts.terminalLines)})
	}

	// Add tmate server overrides only if provided (tmate.py will use these if CLOUD_URL not in its CONFIG)
	if opts.tmateServerHost != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TMATE_SERVER_HOST", Value: opts.tmateServerHost})
	}
	if opts.tmateServerPort != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TMATE_SERVER_PORT", Value: opts.tmateServerPort})
	}
	if opts.tmateServerRSA != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TMATE_SERVER_RSA_FINGERPRINT", Value: opts.tmateServerRSA})
	}
	if opts.tmateServerEd25519 != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TMATE_SERVER_ED25519_FINGERPRINT", Value: opts.tmateServerEd25519})
	}
	if opts.tmateServerECDSA != "" {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "TMATE_SERVER_ECDSA_FINGERPRINT", Value: opts.tmateServerECDSA})
	}

	if opts.debug {
		tmateEnv = append(tmateEnv, corev1.EnvVar{Name: "DEBUG", Value: "1"})
	}

	// Volume mounts for tmate container
	tmateMounts := []corev1.VolumeMount{
		{Name: "shared-socket", MountPath: "/shared"},
		{Name: "ssh-keys", MountPath: "/root/.ssh", ReadOnly: true},
	}

	// Recorder environment
	recorderEnv := []corev1.EnvVar{
		{Name: "SHARED_SOCKET", Value: sharedSocketPath},
		{Name: "CLUSTER_ID", Value: opts.clusterID},
	}

	// Volume mounts for recorder container
	recorderMounts := []corev1.VolumeMount{
		{Name: "shared-socket", MountPath: "/shared"},
		{Name: "recordings", MountPath: "/recordings"},
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("remote-session-%s", sessionID),
			Namespace: chart.ReleaseNamespace,
			Labels: map[string]string{
				remoteAccessLabel: remoteAccessValue,
				"session-id":      sessionID,
				"cluster-id":      opts.clusterID,
				"cluster-name":    opts.clusterName,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:         "tmate",
					Image:        remoteAccessImage,
					Command:      []string{"python3", "/tmate/tmate.py"},
					Env:          tmateEnv,
					VolumeMounts: tmateMounts,
				},
				{
					Name:         "recorder",
					Image:        remoteAccessImage,
					Command:      []string{"python3", "/recorder/recorder.py"},
					Env:          recorderEnv,
					VolumeMounts: recorderMounts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "shared-socket",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "recordings",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: recordingsPVCName,
						},
					},
				},
				{
					Name: "ssh-keys",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: opts.sshKeysPath,
						},
					},
				},
			},
		},
	}
}
