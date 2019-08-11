package list

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/nicerobot/ssh-tgzx/internal/app"
	"github.com/nicerobot/ssh-tgzx/internal/archive"
	"github.com/nicerobot/ssh-tgzx/internal/constants"
	"github.com/nicerobot/ssh-tgzx/internal/crypt"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestListCommand_MissingArgs(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	var stdout bytes.Buffer
	logger := testLogger()

	testApp := &cli.App{
		Name:      "app",
		Writer:    &stdout,
		ErrWriter: os.Stderr,
		Commands: []*cli.Command{
			Command(),
		},
		Metadata: map[string]any{
			app.LoggerMetadataKey: logger,
		},
	}

	err := testApp.RunContext(context.Background(), []string{"app", "list"})
	must.Error(err)
	want.ErrorIs(err, constants.ErrMissingArgument)
}

func TestListCommand_RoundTrip(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	// Generate key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	must.NoError(err)

	sshPub, err := ssh.NewPublicKey(pub)
	must.NoError(err)
	rcpt, err := agessh.ParseRecipient(string(ssh.MarshalAuthorizedKey(sshPub)))
	must.NoError(err)

	privKey, err := ssh.MarshalPrivateKey(priv, "")
	must.NoError(err)

	// Write identity file
	identityFile := filepath.Join(t.TempDir(), "id_ed25519")
	must.NoError(os.WriteFile(identityFile, pem.EncodeToMemory(privKey), 0o600))

	// Create source file
	srcDir := t.TempDir()
	must.NoError(os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("data"), 0o644))

	// Create encrypted archive
	archiveFile := filepath.Join(t.TempDir(), "test.age")
	f, err := os.Create(archiveFile)
	must.NoError(err)

	var archiveBuf bytes.Buffer
	must.NoError(archive.Create(&archiveBuf, []string{filepath.Join(srcDir, "data.txt")}))
	must.NoError(crypt.Encrypt(f, &archiveBuf, []age.Recipient{rcpt}))
	f.Close()

	// List
	logger := testLogger()
	result, err := Run(context.Background(), logger, Config{}, archiveFile, identityFile)
	must.NoError(err)
	want.Greater(result.Count, 0)
	want.NotEmpty(result.Entries)
}
