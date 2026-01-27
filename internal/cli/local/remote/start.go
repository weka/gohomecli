package remote

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/utils"
)

// Kubernetes label value constraints.
const maxLabelValueLength = 63

// labelValuePattern matches valid Kubernetes label values.
// Must start and end with alphanumeric, can contain alphanumerics, dashes, underscores, and dots.
var labelValuePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)

var (
	// ErrSSHKeysPathNotDirectory is returned when the ssh-keys-path exists but is not a directory.
	ErrSSHKeysPathNotDirectory = errors.New("ssh-keys-path must be a directory")

	// ErrClusterNameTooLong is returned when the cluster name exceeds the Kubernetes label value limit.
	ErrClusterNameTooLong = errors.New("cluster-name exceeds maximum length of 63 characters")

	// ErrClusterNameInvalid is returned when the cluster name contains invalid characters for a Kubernetes label.
	ErrClusterNameInvalid = errors.New(
		"cluster-name must start and end with alphanumeric characters, " +
			"and contain only alphanumerics, dashes, underscores, or dots",
	)
)

const (
	remoteAccessLabel         = "app"
	remoteAccessValue         = "remote-access"
	remoteAccessConfigLabel   = "app=remote-access-config"
	sessionRecordingsPVCLabel = "app=remote-access-recordings"
	sessionIDBytes            = 3 // 3 bytes = 6 hex chars
	sharedSocketPath          = "/shared/tmate.socket"
)

type startOptions struct {
	// Tmate server flags (optional overrides - tmate.py has built-in config for known cloud URLs)
	tmateServerHost    string
	tmateServerPort    string
	tmateServerRSA     string
	tmateServerEd25519 string
	tmateServerECDSA   string
	clusterID          string
	clusterName        string
	sshKeysPath        string
	cloudURL           string
	hostName           string
	terminalCols       int
	terminalLines      int
	debug              bool
}

func newStartCmd() *cobra.Command {
	opts := &startOptions{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new remote-access tmate session",
		Long: `Start a remote-access tmate session for remote cluster connection.
Tmate server config is resolved from cloud URL. Use --tmate-server-* flags for custom servers.

Examples:
  homecli remote-access start --cluster-id "550e8400-..." --cluster-name "prod" --ssh-keys-path "/root/.ssh" # Start a session with cloud URL from config
  homecli remote-access start --cluster-id "550e8400-..." --cluster-name "prod" --ssh-keys-path "/root/.ssh" --tmate-server-host "tmate.example.com" --tmate-server-port "22" --tmate-server-rsa-fingerprint "1234567890" --tmate-server-ed25519-fingerprint "1234567890" --tmate-server-ecdsa-fingerprint "1234567890" # Start a session with custom tmate server
			`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return startRun(cmd, opts)
		},
	}

	// Required flags
	cmd.Flags().StringVar(&opts.clusterID, "cluster-id", "", "Cluster GUID (required)")
	cmd.Flags().StringVar(&opts.clusterName, "cluster-name", "", "Human-readable cluster name, max 63 chars, alphanumeric with dashes/underscores/dots (required)")
	cmd.Flags().StringVar(&opts.sshKeysPath, "ssh-keys-path", "", "Host path to existing SSH keys directory, mounted as HostPath volume (required)")
	_ = cmd.MarkFlagRequired("cluster-id")    //nolint:errcheck // flag exists
	_ = cmd.MarkFlagRequired("cluster-name")  //nolint:errcheck // flag exists
	_ = cmd.MarkFlagRequired("ssh-keys-path") //nolint:errcheck // flag exists

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
	cmd.Flags().StringVar(
		&opts.tmateServerEd25519, "tmate-server-ed25519-fingerprint", "", "Override Ed25519 fingerprint",
	)
	cmd.Flags().StringVar(&opts.tmateServerECDSA, "tmate-server-ecdsa-fingerprint", "", "Override ECDSA fingerprint")

	return cmd
}

func startRun(cmd *cobra.Command, opts *startOptions) error {
	ctx := cmd.Context()

	// Validate SSH keys path exists and is a directory
	if err := validateSSHKeysPath(opts.sshKeysPath); err != nil {
		return err
	}

	// Validate cluster name for Kubernetes label compatibility
	if err := validateClusterName(opts.clusterName); err != nil {
		return err
	}

	// Resolve cloud URL from flag, config, or default
	cloudURL := opts.cloudURL
	if cloudURL == "" && env.CurrentSiteConfig != nil {
		cloudURL = env.CurrentSiteConfig.CloudURL
	}
	if cloudURL == "" {
		cloudURL = env.DefaultCloudURL
		utils.UserNote("Cloud URL not configured, using default: %s", cloudURL)
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
	k8s, err := chart.NewKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Get recordings PVC name
	recordingsPVCName, err := chart.GetPVCName(ctx, sessionRecordingsPVCLabel)
	if err != nil {
		return fmt.Errorf("failed to find recordings PVC: %w", err)
	}

	// Get remote-access image from ConfigMap
	remoteAccessImage, err := chart.GetConfigMapData(ctx, remoteAccessConfigLabel, "image")
	if err != nil {
		return fmt.Errorf("failed to get remote-access image from config: %w", err)
	}

	// Create the pod
	pod := buildSessionPod(sessionID, webhookURLs, cloudURL, recordingsPVCName, remoteAccessImage, opts)

	logger.Info().
		Str("sessionID", sessionID).
		Str("clusterID", opts.clusterID).
		Str("clusterName", opts.clusterName).
		Msg("Creating remote session pod...")

	createdPod, err := k8s.Clientset.CoreV1().Pods(chart.ReleaseNamespace).Create(ctx, pod, metav1.CreateOptions{})
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

// validateSSHKeysPath validates that the SSH keys path exists and is a directory.
func validateSSHKeysPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ssh-keys-path does not exist: %s", path)
		}

		return fmt.Errorf("failed to access ssh-keys-path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrSSHKeysPathNotDirectory, path)
	}

	return nil
}

// validateClusterName validates that the cluster name is valid for use as a Kubernetes label value.
func validateClusterName(name string) error {
	if len(name) > maxLabelValueLength {
		return fmt.Errorf("%w (got %d characters)", ErrClusterNameTooLong, len(name))
	}

	// Single character names are valid if alphanumeric
	if len(name) == 1 {
		if !((name[0] >= 'a' && name[0] <= 'z') ||
			(name[0] >= 'A' && name[0] <= 'Z') ||
			(name[0] >= '0' && name[0] <= '9')) {
			return fmt.Errorf("%w: %q", ErrClusterNameInvalid, name)
		}

		return nil
	}

	if !labelValuePattern.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrClusterNameInvalid, name)
	}

	return nil
}

func generateShortID() string {
	b := make([]byte, sessionIDBytes)
	_, _ = rand.Read(b) //nolint:errcheck // crypto/rand.Read error indicates serious system issue

	return hex.EncodeToString(b)
}

func buildTmateEnv(webhookURLs, cloudURL string, opts *startOptions) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "CLUSTER_ID", Value: opts.clusterID},
		{Name: "CLUSTER_NAME", Value: opts.clusterName},
		{Name: "SHARED_SOCKET", Value: sharedSocketPath},
		{Name: "CLOUD_URL", Value: cloudURL},
		{Name: "WEBHOOK_URLS", Value: webhookURLs},
	}

	if opts.hostName != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "HOST_NAME", Value: opts.hostName})
	}
	if opts.terminalCols > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "TERMINAL_COLS", Value: strconv.Itoa(opts.terminalCols)})
	}
	if opts.terminalLines > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "TERMINAL_LINES", Value: strconv.Itoa(opts.terminalLines)})
	}
	if opts.tmateServerHost != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "TMATE_SERVER_HOST", Value: opts.tmateServerHost})
	}
	if opts.tmateServerPort != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "TMATE_SERVER_PORT", Value: opts.tmateServerPort})
	}
	if opts.tmateServerRSA != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "TMATE_SERVER_RSA_FINGERPRINT", Value: opts.tmateServerRSA})
	}
	if opts.tmateServerEd25519 != "" {
		envVars = append(
			envVars, corev1.EnvVar{Name: "TMATE_SERVER_ED25519_FINGERPRINT", Value: opts.tmateServerEd25519},
		)
	}
	if opts.tmateServerECDSA != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "TMATE_SERVER_ECDSA_FINGERPRINT", Value: opts.tmateServerECDSA})
	}
	if opts.debug {
		envVars = append(envVars, corev1.EnvVar{Name: "DEBUG", Value: "1"})
	}

	return envVars
}

func buildPodVolumes(recordingsPVCName, sshKeysPath string) []corev1.Volume {
	return []corev1.Volume{
		{Name: "shared-socket", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "recordings", VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: recordingsPVCName},
		}},
		{Name: "ssh-keys", VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{Path: sshKeysPath},
		}},
		{Name: "dev-pts", VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{Path: "/dev/pts"},
		}},
	}
}

func buildSessionPod(
	sessionID, webhookURLs, cloudURL, recordingsPVCName, image string,
	opts *startOptions,
) *corev1.Pod {
	tmateEnv := buildTmateEnv(webhookURLs, cloudURL, opts)
	recorderEnv := []corev1.EnvVar{
		{Name: "SHARED_SOCKET", Value: sharedSocketPath},
		{Name: "CLUSTER_ID", Value: opts.clusterID},
	}

	tmateMounts := []corev1.VolumeMount{
		{Name: "shared-socket", MountPath: "/shared"},
		{Name: "ssh-keys", MountPath: "/root/.ssh", ReadOnly: true},
		{Name: "dev-pts", MountPath: "/dev/pts"},
	}
	recorderMounts := []corev1.VolumeMount{
		{Name: "shared-socket", MountPath: "/shared"},
		{Name: "recordings", MountPath: "/recordings"},
		{Name: "dev-pts", MountPath: "/dev/pts"},
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "remote-session-" + sessionID,
			Namespace: chart.ReleaseNamespace,
			Labels: map[string]string{
				remoteAccessLabel: remoteAccessValue,
				"session-id":      sessionID,
				"cluster-id":      opts.clusterID,
				"cluster-name":    opts.clusterName,
			},
		},
		Spec: corev1.PodSpec{
			ShareProcessNamespace: ptr.To(true),
			RestartPolicy:         corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:         "tmate",
					Image:        image,
					Command:      []string{"python3", "/tmate/tmate.py"},
					Env:          tmateEnv,
					VolumeMounts: tmateMounts,
				},
				{
					Name:         "recorder",
					Image:        image,
					Command:      []string{"python3", "/recorder/recorder.py"},
					Env:          recorderEnv,
					VolumeMounts: recorderMounts,
				},
			},
			Volumes: buildPodVolumes(recordingsPVCName, opts.sshKeysPath),
		},
	}
}
