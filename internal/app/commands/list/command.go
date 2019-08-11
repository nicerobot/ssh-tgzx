package list

import (
	"bytes"
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/nicerobot/ssh-tgzx/internal/app"
	"github.com/nicerobot/ssh-tgzx/internal/archive"
	"github.com/nicerobot/ssh-tgzx/internal/constants"
	"github.com/nicerobot/ssh-tgzx/internal/crypt"
)

const (
	name        = `list`
	usage       = `List contents of an encrypted archive.`
	argUsage    = `<archive-file> <identity-file>`
	description = `Decrypt an age-encrypted tar.gz archive and list its contents without extracting.`
)

// Config holds the configuration for the list command.
type Config struct{}

// Result holds the output of the list command.
type Result struct {
	Entries []string `json:"entries"`
	Count   int      `json:"count"`
}

var (
	cfg       Config
	runAction = Run
)

// Command returns the CLI command definition.
func Command() *cli.Command {
	return &cli.Command{
		Name:        name,
		Usage:       usage,
		ArgsUsage:   argUsage,
		Description: description,
		Action:      app.Default(&cfg, runAction),
	}
}

// Run executes the list command.
func Run(ctx context.Context, logger *slog.Logger, config Config, args ...string) (Result, error) {
	if len(args) < 2 {
		return Result{}, constants.ErrMissingArgument.Wrap(nil, "usage: <archive-file> <identity-file>")
	}

	archiveFile := args[0]
	identityFile := args[1]

	identities, err := crypt.ParseIdentities(identityFile)
	if err != nil {
		return Result{}, err
	}

	f, err := os.Open(archiveFile)
	if err != nil {
		return Result{}, constants.ErrOpenFile.Wrap(err, archiveFile)
	}
	defer func() { _ = f.Close() }()

	var decrypted bytes.Buffer
	if err := crypt.Decrypt(&decrypted, f, identities); err != nil {
		return Result{}, err
	}

	entries, err := archive.List(&decrypted)
	if err != nil {
		return Result{}, err
	}

	logger.Info("Listed archive", "file", archiveFile, "count", len(entries))

	return Result{
		Entries: entries,
		Count:   len(entries),
	}, nil
}
