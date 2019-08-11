package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestRun_Version(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{
			name:         "version flag outputs version",
			args:         []string{"ssh-tgzx", "--version"},
			wantContains: version,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			var stdout bytes.Buffer

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))

			app := createApp(func(c *cli.Context) *slog.Logger { return logger })
			app.Writer = &stdout

			err := app.RunContext(context.Background(), tt.args)
			must.NoError(err)

			output := stdout.String()
			want.Contains(output, tt.wantContains)
		})
	}
}

func TestCreateApp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		expectedName     string
		expectedVersion  string
		expectedCommands []string
	}{
		{
			name:             "creates app with correct name and version",
			expectedName:     name,
			expectedVersion:  version,
			expectedCommands: []string{"create", "extract", "list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			app := createApp(func(c *cli.Context) *slog.Logger { return logger })

			want.Equal(tt.expectedName, app.Name)
			want.Equal(tt.expectedVersion, app.Version)
			must.NotEmpty(app.Commands, "expected app to have commands")

			for _, expected := range tt.expectedCommands {
				found := false
				for _, cmd := range app.Commands {
					if cmd.Name == expected {
						found = true
						break
					}
				}
				want.True(found, "expected command %q not found", expected)
			}
		})
	}
}
