package create

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"

	"filippo.io/age"
	"github.com/urfave/cli/v2"

	"github.com/nicerobot/ssh-tgzx/internal/app"
	"github.com/nicerobot/ssh-tgzx/internal/archive"
	"github.com/nicerobot/ssh-tgzx/internal/constants"
	"github.com/nicerobot/ssh-tgzx/internal/crypt"
	"github.com/nicerobot/ssh-tgzx/internal/ghkeys"
)

const (
	name        = `create`
	usage       = `Create an encrypted archive for a GitHub user.`
	argUsage    = `<github-username> <archive-file> <paths...>`
	description = `Create an age-encrypted tar.gz archive secured with the SSH public keys
of the specified GitHub user. The recipient can decrypt it using their
SSH private key with the extract command.`
)

// KeyFetcher is the function type for fetching age recipients.
type KeyFetcher func(ctx context.Context, client ghkeys.HTTPClient, username string) ([]age.Recipient, error)

// Config holds the configuration for the create command.
type Config struct {
	KeyFetcher KeyFetcher `json:"-"`
}

// Result holds the output of the create command.
type Result struct {
	File       string `json:"file"`
	Recipients int    `json:"recipients"`
	Size       int64  `json:"size"`
}

var (
	cfg       Config
	runAction = Run
)

func init() {
	cfg.KeyFetcher = ghkeys.FetchRecipients
}

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

// Run executes the create command.
func Run(ctx context.Context, logger *slog.Logger, config Config, args ...string) (Result, error) {
	if len(args) < 3 {
		return Result{}, constants.ErrMissingArgument.Wrap(nil, "usage: <github-username> <archive-file> <paths...>")
	}

	username := args[0]
	archiveFile := args[1]
	paths := args[2:]

	fetcher := config.KeyFetcher
	if fetcher == nil {
		fetcher = ghkeys.FetchRecipients
	}

	recipients, err := fetcher(ctx, http.DefaultClient, username)
	if err != nil {
		return Result{}, err
	}

	logger.Info("Fetched recipients", "username", username, "count", len(recipients))

	f, err := os.Create(archiveFile)
	if err != nil {
		return Result{}, constants.ErrOpenFile.Wrap(err, archiveFile)
	}
	defer func() { _ = f.Close() }()

	// Pipe: archive creation -> age encryption -> output file
	pr, pw := io.Pipe()

	errCh := make(chan error, 1)
	go func() {
		err := archive.Create(pw, paths)
		_ = pw.CloseWithError(err)
		errCh <- err
	}()

	if err := crypt.Encrypt(f, pr, recipients); err != nil {
		return Result{}, err
	}

	if err := <-errCh; err != nil {
		return Result{}, err
	}

	info, err := f.Stat()
	if err != nil {
		return Result{}, err
	}

	return Result{
		File:       archiveFile,
		Recipients: len(recipients),
		Size:       info.Size(),
	}, nil
}
