package k3s

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/weka/gohomecli/internal/utils"
)

var tlsYaml = `---
apiVersion: traefik.containo.us/v1alpha1
kind: TLSStore
metadata:
  name: default
  namespace: kube-system
spec:
  defaultCertificate:
    secretName: tls-secret
`

var ErrNoTLS = errors.New("no tls files")

type TLSConfig struct {
	CertFile string
	KeyFile  string
}

func (t *TLSConfig) WithDefaults() TLSConfig {
	if t.CertFile == "" {
		t.CertFile = "/etc/ssl/certs/k3s.cert"
	}
	if t.KeyFile == "" {
		t.KeyFile = "/etc/ssl/k3s.pem"
	}
	return *t
}

func setupTLS(ctx context.Context, config TLSConfig) error {
	config = config.WithDefaults()

	if !utils.IsFileExists(config.CertFile) || !utils.IsFileExists(config.KeyFile) {
		logger.Warn().Msg("No TLS configuration added")
		return ErrNoTLS
	}

	logger.Info().Msg("Adding TLS secret")

	cmd, err := utils.ExecCommand(ctx, "kubectl", []string{
		"create", "secret", "tls", "tls-secret",
		"--namespace", "kube-system", "--cert", config.CertFile, "--key", config.KeyFile,
	}, utils.WithStderrLogger(logger, utils.WarnLevel), utils.WithStdoutLogger(logger, utils.InfoLevel))

	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("kubectl create secret: %w", err)
	}

	logger.Info().Msg("Waiting for Traefik to be ready")

	waitScript := `until [[ $(kubectl get endpoints/traefik -n kube-system) ]]; do sleep 5; done`

	cmd, err = utils.ExecCommand(ctx, "bash", []string{"-"},
		utils.WithStdin(strings.NewReader(waitScript)),
		utils.WithStderrLogger(logger, utils.DebugLevel))
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl wait: %w", err)
	}

	logger.Info().Msg("Applying traefik config")
	cmd, err = utils.ExecCommand(ctx, "kubectl", []string{"apply", "-f", "-"},
		utils.WithStdin(strings.NewReader(tlsYaml)),
		utils.WithStdoutLogger(logger, utils.InfoLevel), utils.WithStderrLogger(logger, utils.WarnLevel))
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl apply config: %w", err)
	}

	logger.Info().Msg("TLS was configured successfully")
	return nil
}
