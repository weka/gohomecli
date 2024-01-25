package k3s

import (
	"context"
	"errors"
	"fmt"

	"github.com/weka/gohomecli/internal/utils"
)

var openPorts = []int{80, 443, 6443, 10250, 10257, 10259}

var openNetworks = []string{
	"10.42.0.0/16", // pods
	"10.43.0.0/16", // services
}

func isFirewalldActive(ctx context.Context) bool {
	logger.Info().Msg("Checking if firewalld is enabled")

	var active bool

	cmd, err := utils.ExecCommand(ctx, "systemctl",
		[]string{"is-active", "firewalld"},
		utils.WithStdoutReader(func(lines chan []byte) {
			for line := range lines {
				logger.Debug().Str("output", string(line)).Msg("firewalld status")
				if string(line) == "active" {
					active = true
				}
			}
		}))

	err = errors.Join(err, cmd.Wait())
	if err != nil && cmd.ProcessState.ExitCode() != 3 {
		logger.Debug().Err(err).Msg("systemctl exit status")
	}

	return active
}

func isUFWActive(ctx context.Context) bool {
	logger.Info().Msg("Checking if UFW is enabled")

	var active bool

	cmd, err := utils.ExecCommand(ctx, "systemctl",
		[]string{"is-active", "ufw"},
		utils.WithStdoutReader(func(lines chan []byte) {
			for line := range lines {
				logger.Debug().Str("output", string(line)).Msg("ufw enabled")
				if string(line) == "active" {
					active = true
				}
			}
		}))

	err = errors.Join(err, cmd.Wait())
	if err != nil && cmd.ProcessState.ExitCode() != 3 {
		logger.Debug().Err(err).Msg("systemctl exit status")
	}

	return active
}

func addFirewalldRules(ctx context.Context) error {
	logger.Info().Msg("Adding firewalld rules")

	for _, port := range openPorts {
		args := []string{
			"--add-port", fmt.Sprintf("%d/tcp", port), "--permanent",
		}

		cmd, err := utils.ExecCommand(ctx, "firewall-cmd", args,
			utils.WithStderrReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("error adding firewalld rule %v", args)
				}
			}),
			utils.WithStdoutReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("firewalld rule %v added", args)
				}
			}),
		)

		err = errors.Join(err, cmd.Wait())
		if err != nil {
			return err
		}
	}

	for _, network := range openNetworks {
		args := []string{"--add-source", network, "--permanent", "--zone=trusted"}
		cmd, err := utils.ExecCommand(ctx, "firewall-cmd", args,
			utils.WithStderrReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("error adding firewalld rule %v", args)
				}
			}),
			utils.WithStdoutReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("firewalld rule %v added", args)
				}
			}),
		)
		err = errors.Join(err, cmd.Wait())
		if err != nil {
			return err
		}
	}

	cmd, err := utils.ExecCommand(ctx, "firewall-cmd", []string{"--reload"})
	err = errors.Join(err, cmd.Wait())
	if err != nil {
		return err
	}

	return nil
}

func addUFWRules(ctx context.Context) error {
	logger.Info().Msg("Adding UFW rules")

	for _, port := range openPorts {
		args := []string{
			"allow", fmt.Sprintf("%d/tcp", port),
		}

		cmd, err := utils.ExecCommand(ctx, "ufw", args,
			utils.WithStderrReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("error adding UFW rule %v", args)
				}
			}),
			utils.WithStdoutReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("UFW rule %v added", args)
				}
			}),
		)

		err = errors.Join(err, cmd.Wait())
		if err != nil {
			return err
		}
	}

	for _, network := range openNetworks {
		args := []string{"allow", "from", network, "to", "any"}
		cmd, err := utils.ExecCommand(ctx, "ufw", args,
			utils.WithStderrReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("error adding UFW rule %v", args)
				}
			}),
			utils.WithStdoutReader(func(lines chan []byte) {
				for line := range lines {
					logger.Debug().Str("output", string(line)).Msgf("UFW rule %v added", args)
				}
			}),
		)

		if err = errors.Join(err, cmd.Wait()); err != nil {
			return err
		}
	}

	return nil
}
