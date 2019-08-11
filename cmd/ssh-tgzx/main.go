package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sort"

	"github.com/urfave/cli/v2"

	"github.com/nicerobot/ssh-tgzx/internal/app"
	"github.com/nicerobot/ssh-tgzx/internal/app/commands/create"
	"github.com/nicerobot/ssh-tgzx/internal/app/commands/extract"
	"github.com/nicerobot/ssh-tgzx/internal/app/commands/list"
)

const (
	argUsage    = ``
	description = `Create and extract age-encrypted tar.gz archives secured with SSH keys.

Archives are encrypted using the SSH public keys of a GitHub user.
Recipients decrypt using their SSH private key.

Supported key types: RSA, Ed25519.

Available Commands:
  create   - Create an encrypted archive for a GitHub user
  extract  - Decrypt and extract an archive
  list     - List contents of an encrypted archive`
	envName   = "SSH_TGZX"
	envPrefix = envName + "_"
	name      = `ssh-tgzx`
	usage     = `Create and extract age-encrypted archives using SSH keys.`
)

var (
	appCreator    = createApp
	loggerConfig  app.LoggerConfig
	loggerCreator = app.GetLogger
)

type (
	appVersion string
)

// version is the application version.
// Set via ldflags: -X main.version=1.0.0
var version = "dev"

func getVersion() appVersion {
	return appVersion(version)
}

func main() { run() }

func run() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	c := appCreator(loggerCreator)

	if err := c.RunContext(ctx, os.Args); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

// createApp constructs the definition of the CLI.
func createApp(getLogger app.GetLoggerFunc) *cli.App {
	cliApp := &cli.App{
		Name:                 name,
		Usage:                usage,
		ArgsUsage:            argUsage,
		Description:          description,
		Version:              string(getVersion()),
		EnableBashCompletion: true,
		Commands: cli.Commands{
			create.Command(),
			extract.Command(),
			list.Command(),
		},
		Before: func(c *cli.Context) error {
			c.App.Metadata[app.LoggerMetadataKey] = getLogger(c)
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				EnvVars:     []string{envPrefix + "LOG_LEVEL"},
				Value:       "info",
				Usage:       "Set the logging level (debug, info, warn, error)",
				Destination: (*string)(&loggerConfig.LogLevel),
			},
			&cli.StringFlag{
				Name:        "log-format",
				EnvVars:     []string{envPrefix + "LOG_FORMAT"},
				Value:       "text",
				Usage:       "Set the log output format (text, json)",
				Destination: (*string)(&loggerConfig.LogFormat),
			},
		},
	}

	sort.Sort(cli.FlagsByName(cliApp.Flags))
	sort.Sort(cli.CommandsByName(cliApp.Commands))

	return cliApp
}
