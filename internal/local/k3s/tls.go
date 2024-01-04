package k3s

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"text/template"

	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

const waitScript = `until [[ $(kubectl get endpoints/traefik -n kube-system) ]]; do sleep 5; done`

var tlsYaml = `---
apiVersion: traefik.containo.us/v1alpha1
kind: TLSStore
metadata:
  name: default
  namespace: kube-system
spec:
  defaultCertificate:
    secretName: tls-secret
---
apiVersion: v1
kind: Secret
metadata:
  name: tls-secret
  namespace: kube-system
data:
  tls.crt: {{ .Cert | b64enc }}
  tls.key: {{ .Key | b64enc }}
type: Opaque
`

func init() {
	tlsTemplate, _ = template.New("tls").
		Funcs(template.FuncMap{
			"b64enc": func(s string) string {
				return base64.StdEncoding.EncodeToString([]byte(s))
			},
		}).
		Parse(tlsYaml)
}

var (
	ErrNoTLS    = errors.New("no tls files")
	tlsTemplate *template.Template
)

func setupTLS(ctx context.Context, config config_v1.Configuration) error {
	if config.TLS.Key == "" || config.TLS.Cert == "" {
		logger.Warn().Msg("No TLS configuration added")
		return ErrNoTLS
	}

	logger.Info().Msg("Adding TLS secret")

	logger.Info().Msg("Waiting for Traefik to be ready")

	cmd, err := utils.ExecCommand(ctx, "bash",
		[]string{"-"},
		utils.WithStdin(strings.NewReader(waitScript)),
		utils.WithStderrLogger(logger, utils.DebugLevel),
	)
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl wait: %w", err)
	}

	var buf bytes.Buffer

	if err := tlsTemplate.Execute(&buf, config.TLS); err != nil {
		return fmt.Errorf("TLS template: %w", err)
	}

	logger.Info().Msg("Applying TLS config")
	cmd, err = utils.ExecCommand(ctx, "kubectl",
		[]string{"apply", "-f", "-"},
		utils.WithStdin(&buf),
		utils.WithStdoutLogger(logger, utils.InfoLevel),
		utils.WithStderrLogger(logger, utils.WarnLevel),
	)
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl apply config: %w", err)
	}

	logger.Info().Msg("TLS was configured successfully")
	return nil
}
