package k3s

import (
	"context"
	"errors"
	"fmt"

	"github.com/weka/gohomecli/internal/utils"
)

type FirewallType string

var (
	FirewallTypeFirewalld FirewallType = "firewalld"
	FirewallTypeUFW       FirewallType = "ufw"
)

var openPorts = []int{80, 443, 6443, 10250, 10257, 10259}

var openNetworks = []string{
	"10.42.0.0/16", // pods
	"10.43.0.0/16", // services
}

func isFirewallActive(ctx context.Context, fw FirewallType) bool {
	logger.Info().Msgf("Checking if %s is enabled", fw)

	var active bool

	cmd, err := utils.ExecCommand(ctx, "systemctl",
		[]string{"is-active", string(fw)},
		utils.WithStderrLogger(logger, utils.DebugLevel),
		utils.WithStdoutReader(func(lines chan []byte) {
			for line := range lines {
				logger.Debug().Str("output", string(line)).Msgf("%s status", fw)
				if string(line) == "active" {
					active = true
				}
			}
		}))

	err = errors.Join(err, cmd.Wait())
	if err != nil && cmd.ProcessState.ExitCode() != 3 { // 3 means no systemd unit exists
		logger.Debug().Err(err).Msg("systemctl exit status")
	}

	return active
}

func addFirewallRules(ctx context.Context, fw FirewallType) error {
	logger.Info().Msgf("Adding firewall rules for %s", fw)

	cmd, rules := firewallRules(fw)
	if cmd == "" {
		return errors.New("unsupported firewall")
	}

	for _, args := range rules {
		cmd, err := utils.ExecCommand(ctx, cmd, args,
			utils.WithStderrLogger(logger, utils.DebugLevel),
			utils.WithStdoutLogger(logger, utils.DebugLevel),
		)

		err = errors.Join(err, cmd.Wait())
		if err != nil {
			return err
		}
		logger.Info().Strs("args", args).Msgf("Added %s rule", cmd)
	}

	if fw == FirewallTypeFirewalld {
		cmd, err := utils.ExecCommand(ctx, "firewall-cmd", []string{"--reload"})
		err = errors.Join(err, cmd.Wait())
		if err != nil {
			return err
		}
	}

	return nil
}

func firewallRules(fw FirewallType) (cmd string, rules [][]string) {
	switch fw {
	case FirewallTypeFirewalld:
		cmd = "firewall-cmd"
	case FirewallTypeUFW:
		cmd = "ufw"
	}

	for _, port := range openPorts {
		switch fw {
		case FirewallTypeFirewalld:
			rules = append(rules, []string{
				"--add-port", fmt.Sprintf("%d/tcp", port), "--permanent",
			})

		case FirewallTypeUFW:
			rules = append(rules, []string{
				"allow", fmt.Sprintf("%d/tcp", port),
			})
		}
	}

	for _, network := range openNetworks {
		switch fw {
		case FirewallTypeFirewalld:
			rules = append(rules, []string{
				"--add-source", network, "--permanent", "--zone=trusted",
			})

		case FirewallTypeUFW:
			rules = append(rules, []string{
				"allow", "from", network, "to", "any",
			})
		}
	}

	return cmd, rules
}
