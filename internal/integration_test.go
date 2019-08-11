//go:build integration

package internal

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
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

	"github.com/nicerobot/ssh-tgzx/internal/archive"
	"github.com/nicerobot/ssh-tgzx/internal/crypt"
	"github.com/nicerobot/ssh-tgzx/internal/ghkeys"
)

func TestIntegration_RoundTrip_Ed25519(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	// Generate ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	must.NoError(err)

	sshPub, err := ssh.NewPublicKey(pub)
	must.NoError(err)
	pubKeyStr := string(ssh.MarshalAuthorizedKey(sshPub))

	privKey, err := ssh.MarshalPrivateKey(priv, "")
	must.NoError(err)
	privPEM := pem.EncodeToMemory(privKey)

	// Start mock GitHub server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pubKeyStr))
	}))
	defer srv.Close()

	// Fetch recipients via ghkeys
	client := &rewriteClient{base: srv.Client(), targetURL: srv.URL}
	recipients, err := ghkeys.FetchRecipients(context.Background(), client, "testuser")
	must.NoError(err)
	must.Len(recipients, 1)

	// Create source files
	srcDir := t.TempDir()
	must.NoError(os.WriteFile(filepath.Join(srcDir, "secret.txt"), []byte("top secret ed25519"), 0o644))
	must.NoError(os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755))
	must.NoError(os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested secret"), 0o644))

	// Create archive -> encrypt
	archiveFile := filepath.Join(t.TempDir(), "test.age")
	f, err := os.Create(archiveFile)
	must.NoError(err)

	var archiveBuf bytes.Buffer
	must.NoError(archive.Create(&archiveBuf, []string{srcDir}))
	must.NoError(crypt.Encrypt(f, &archiveBuf, recipients))
	f.Close()

	// Parse identity
	identityFile := filepath.Join(t.TempDir(), "id_ed25519")
	must.NoError(os.WriteFile(identityFile, privPEM, 0o600))

	identities, err := crypt.ParseIdentities(identityFile)
	must.NoError(err)

	// Decrypt -> extract
	ef, err := os.Open(archiveFile)
	must.NoError(err)
	defer ef.Close()

	var decrypted bytes.Buffer
	must.NoError(crypt.Decrypt(&decrypted, ef, identities))

	extractDir := t.TempDir()
	files, err := archive.Extract(&decrypted, extractDir)
	must.NoError(err)
	want.NotEmpty(files)

	// Verify content
	data, err := os.ReadFile(filepath.Join(extractDir, srcDir, "secret.txt"))
	must.NoError(err)
	want.Equal("top secret ed25519", string(data))

	data, err = os.ReadFile(filepath.Join(extractDir, srcDir, "subdir", "nested.txt"))
	must.NoError(err)
	want.Equal("nested secret", string(data))
}

func TestIntegration_RoundTrip_RSA(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	// Generate RSA key pair
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	must.NoError(err)

	sshPub, err := ssh.NewPublicKey(&rsaPriv.PublicKey)
	must.NoError(err)
	pubKeyStr := string(ssh.MarshalAuthorizedKey(sshPub))

	privKey, err := ssh.MarshalPrivateKey(rsaPriv, "")
	must.NoError(err)
	privPEM := pem.EncodeToMemory(privKey)

	// Start mock GitHub server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pubKeyStr))
	}))
	defer srv.Close()

	client := &rewriteClient{base: srv.Client(), targetURL: srv.URL}
	recipients, err := ghkeys.FetchRecipients(context.Background(), client, "testuser")
	must.NoError(err)
	must.Len(recipients, 1)

	// Create source file
	srcDir := t.TempDir()
	must.NoError(os.WriteFile(filepath.Join(srcDir, "rsa-secret.txt"), []byte("rsa encrypted data"), 0o644))

	// Create archive -> encrypt
	archiveFile := filepath.Join(t.TempDir(), "rsa-test.age")
	f, err := os.Create(archiveFile)
	must.NoError(err)

	var archiveBuf bytes.Buffer
	must.NoError(archive.Create(&archiveBuf, []string{filepath.Join(srcDir, "rsa-secret.txt")}))
	must.NoError(crypt.Encrypt(f, &archiveBuf, recipients))
	f.Close()

	// Parse identity
	identityFile := filepath.Join(t.TempDir(), "id_rsa")
	must.NoError(os.WriteFile(identityFile, privPEM, 0o600))

	identities, err := crypt.ParseIdentities(identityFile)
	must.NoError(err)

	// Decrypt -> list
	lf, err := os.Open(archiveFile)
	must.NoError(err)

	var decBuf bytes.Buffer
	must.NoError(crypt.Decrypt(&decBuf, lf, identities))
	lf.Close()

	entries, err := archive.List(&decBuf)
	must.NoError(err)
	want.NotEmpty(entries)

	// Decrypt -> extract
	ef, err := os.Open(archiveFile)
	must.NoError(err)
	defer ef.Close()

	var decrypted bytes.Buffer
	must.NoError(crypt.Decrypt(&decrypted, ef, identities))

	extractDir := t.TempDir()
	files, err := archive.Extract(&decrypted, extractDir)
	must.NoError(err)
	want.NotEmpty(files)

	data, err := os.ReadFile(filepath.Join(extractDir, srcDir, "rsa-secret.txt"))
	must.NoError(err)
	want.Equal("rsa encrypted data", string(data))
}

func TestIntegration_MultipleRecipients(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	// Generate two different key pairs
	pub1, priv1, err := ed25519.GenerateKey(rand.Reader)
	must.NoError(err)
	sshPub1, _ := ssh.NewPublicKey(pub1)
	pubStr1 := string(ssh.MarshalAuthorizedKey(sshPub1))

	rsaPriv2, err := rsa.GenerateKey(rand.Reader, 2048)
	must.NoError(err)
	sshPub2, _ := ssh.NewPublicKey(&rsaPriv2.PublicKey)
	pubStr2 := string(ssh.MarshalAuthorizedKey(sshPub2))

	// Both keys served
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pubStr1 + pubStr2))
	}))
	defer srv.Close()

	client := &rewriteClient{base: srv.Client(), targetURL: srv.URL}
	recipients, err := ghkeys.FetchRecipients(context.Background(), client, "testuser")
	must.NoError(err)
	must.Len(recipients, 2)

	// Create source
	srcDir := t.TempDir()
	must.NoError(os.WriteFile(filepath.Join(srcDir, "multi.txt"), []byte("multi-key"), 0o644))

	// Encrypt
	var archiveBuf bytes.Buffer
	must.NoError(archive.Create(&archiveBuf, []string{filepath.Join(srcDir, "multi.txt")}))

	var encrypted bytes.Buffer
	must.NoError(crypt.Encrypt(&encrypted, &archiveBuf, recipients))

	// Either key should decrypt
	for i, priv := range []any{priv1, rsaPriv2} {
		privKey, err := ssh.MarshalPrivateKey(priv, "")
		must.NoError(err)
		privPEM := pem.EncodeToMemory(privKey)

		id, err := agessh.ParseIdentity(privPEM)
		must.NoError(err)

		var dec bytes.Buffer
		err = crypt.Decrypt(&dec, bytes.NewReader(encrypted.Bytes()), []age.Identity{id})
		must.NoError(err, "recipient %d should decrypt", i)
	}
}

type rewriteClient struct {
	base      *http.Client
	targetURL string
}

func (c *rewriteClient) Do(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = c.targetURL[len("http://"):]
	return c.base.Do(req)
}
