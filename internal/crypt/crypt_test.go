package crypt

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateEd25519Identity(t *testing.T) (age.Identity, age.Recipient, []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	rcpt, err := agessh.ParseRecipient(string(ssh.MarshalAuthorizedKey(sshPub)))
	require.NoError(t, err)

	privKey, err := ssh.MarshalPrivateKey(priv, "")
	require.NoError(t, err)

	id, err := agessh.ParseIdentity(pem.EncodeToMemory(privKey))
	require.NoError(t, err)

	return id, rcpt, pem.EncodeToMemory(privKey)
}

func generateRSAIdentity(t *testing.T) (age.Identity, age.Recipient, []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	rcpt, err := agessh.ParseRecipient(string(ssh.MarshalAuthorizedKey(sshPub)))
	require.NoError(t, err)

	privKey, err := ssh.MarshalPrivateKey(priv, "")
	require.NoError(t, err)

	id, err := agessh.ParseIdentity(pem.EncodeToMemory(privKey))
	require.NoError(t, err)

	return id, rcpt, pem.EncodeToMemory(privKey)
}

func TestEncryptDecrypt_Ed25519(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	id, rcpt, _ := generateEd25519Identity(t)

	plaintext := []byte("secret data for ed25519")

	var encrypted bytes.Buffer
	must.NoError(Encrypt(&encrypted, bytes.NewReader(plaintext), []age.Recipient{rcpt}))

	var decrypted bytes.Buffer
	must.NoError(Decrypt(&decrypted, &encrypted, []age.Identity{id}))

	want.Equal(plaintext, decrypted.Bytes())
}

func TestEncryptDecrypt_RSA(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	id, rcpt, _ := generateRSAIdentity(t)

	plaintext := []byte("secret data for rsa")

	var encrypted bytes.Buffer
	must.NoError(Encrypt(&encrypted, bytes.NewReader(plaintext), []age.Recipient{rcpt}))

	var decrypted bytes.Buffer
	must.NoError(Decrypt(&decrypted, &encrypted, []age.Identity{id}))

	want.Equal(plaintext, decrypted.Bytes())
}

func TestDecrypt_WrongKey(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	_, rcpt1, _ := generateEd25519Identity(t)
	id2, _, _ := generateEd25519Identity(t)

	plaintext := []byte("wrong key test")

	var encrypted bytes.Buffer
	must.NoError(Encrypt(&encrypted, bytes.NewReader(plaintext), []age.Recipient{rcpt1}))

	var decrypted bytes.Buffer
	err := Decrypt(&decrypted, &encrypted, []age.Identity{id2})
	must.Error(err)
}

func TestEncryptDecrypt_MultipleRecipients(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	id1, rcpt1, _ := generateEd25519Identity(t)
	id2, rcpt2, _ := generateRSAIdentity(t)

	plaintext := []byte("multi-recipient secret")

	var encrypted1, encrypted2 bytes.Buffer
	// Encrypt for both recipients
	var encrypted bytes.Buffer
	must.NoError(Encrypt(&encrypted, bytes.NewReader(plaintext), []age.Recipient{rcpt1, rcpt2}))

	// Either identity should be able to decrypt
	encrypted1 = *bytes.NewBuffer(encrypted.Bytes())
	encrypted2 = *bytes.NewBuffer(encrypted.Bytes())

	var dec1 bytes.Buffer
	must.NoError(Decrypt(&dec1, &encrypted1, []age.Identity{id1}))
	want.Equal(plaintext, dec1.Bytes())

	var dec2 bytes.Buffer
	must.NoError(Decrypt(&dec2, &encrypted2, []age.Identity{id2}))
	want.Equal(plaintext, dec2.Bytes())
}

func TestParseIdentities(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	_, _, privPEM := generateEd25519Identity(t)

	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	must.NoError(os.WriteFile(keyFile, privPEM, 0o600))

	ids, err := ParseIdentities(keyFile)
	must.NoError(err)
	want.Len(ids, 1)
}

func TestParseIdentities_Nonexistent(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	_, err := ParseIdentities("/nonexistent/path/id_ed25519")
	must.Error(err)
}
