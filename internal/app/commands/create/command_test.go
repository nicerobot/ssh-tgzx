package create

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	"github.com/nicerobot/ssh-tgzx/internal/constants"
	"github.com/nicerobot/ssh-tgzx/internal/ghkeys"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestCreateCommand(t *testing.T) {
	t.Parallel()

	// Generate a test key
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	pubKeyStr := string(ssh.MarshalAuthorizedKey(sshPub))

	// Start a test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(pubKeyStr))
	}))
	defer srv.Close()

	// Create a test fetcher that uses the test server
	testFetcher := func(ctx context.Context, client ghkeys.HTTPClient, username string) ([]age.Recipient, error) {
		rcpt, err := agessh.ParseRecipient(pubKeyStr)
		if err != nil {
			return nil, err
		}
		return []age.Recipient{rcpt}, nil
	}

	tests := []struct {
		name           string
		args           []string
		wantErr        error
		wantOutputCont string
	}{
		{
			name:    "missing arguments",
			args:    []string{"app", "create"},
			wantErr: constants.ErrMissingArgument,
		},
		{
			name:    "missing archive file and paths",
			args:    []string{"app", "create", "testuser"},
			wantErr: constants.ErrMissingArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			var stdout bytes.Buffer
			logger := testLogger()

			localCfg := Config{KeyFetcher: testFetcher}

			testApp := &cli.App{
				Name:      "app",
				Writer:    &stdout,
				ErrWriter: os.Stderr,
				Commands: []*cli.Command{
					{
						Name:   name,
						Action: app.Default(&localCfg, Run),
					},
				},
				Metadata: map[string]any{
					app.LoggerMetadataKey: logger,
				},
			}

			err := testApp.RunContext(context.Background(), tt.args)

			if tt.wantErr != nil {
				must.Error(err)
				want.ErrorIs(err, tt.wantErr)
				return
			}

			must.NoError(err)
			if tt.wantOutputCont != "" {
				want.Contains(stdout.String(), tt.wantOutputCont)
			}
		})
	}
}

func TestCreateCommand_Success(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	// Generate a test key
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	must.NoError(err)
	sshPub, err := ssh.NewPublicKey(pub)
	must.NoError(err)
	pubKeyStr := string(ssh.MarshalAuthorizedKey(sshPub))

	testFetcher := func(ctx context.Context, client ghkeys.HTTPClient, username string) ([]age.Recipient, error) {
		rcpt, err := agessh.ParseRecipient(pubKeyStr)
		if err != nil {
			return nil, err
		}
		return []age.Recipient{rcpt}, nil
	}

	// Create source files
	srcDir := t.TempDir()
	must.NoError(os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0o644))

	outDir := t.TempDir()
	archiveFile := filepath.Join(outDir, "test.age")

	logger := testLogger()

	result, err := Run(context.Background(), logger, Config{KeyFetcher: testFetcher},
		"testuser", archiveFile, filepath.Join(srcDir, "test.txt"))

	must.NoError(err)
	want.Equal(archiveFile, result.File)
	want.Equal(1, result.Recipients)
	want.Greater(result.Size, int64(0))

	// Verify file exists
	info, err := os.Stat(archiveFile)
	must.NoError(err)
	want.Greater(info.Size(), int64(0))
}
