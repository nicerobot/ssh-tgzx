package ghkeys

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"

	"github.com/nicerobot/ssh-tgzx/internal/constants"
)

// HTTPClient is the interface for making HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// FetchRecipients fetches SSH public keys for a GitHub user and returns age recipients.
func FetchRecipients(ctx context.Context, client HTTPClient, username string) ([]age.Recipient, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, constants.ErrFetchKeys.Wrap(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, constants.ErrFetchKeys.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, constants.ErrFetchKeys.Wrap(nil, "HTTP", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, constants.ErrFetchKeys.Wrap(err)
	}

	var recipients []age.Recipient

	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		rcpt, err := agessh.ParseRecipient(line)
		if err != nil {
			slog.Warn("Skipping unsupported key", "key", line[:min(40, len(line))], "error", err)
			continue
		}

		recipients = append(recipients, rcpt)
	}

	if len(recipients) == 0 {
		return nil, constants.ErrNoValidKeys.Wrap(nil, username)
	}

	return recipients, nil
}
