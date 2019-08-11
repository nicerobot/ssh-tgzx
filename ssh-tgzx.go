package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s github-username archive-file [files | directories]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	username := os.Args[1]
	archiveFile := os.Args[2]
	paths := os.Args[3:]

	publicKey, err := fetchGitHubKey(username)
	if err != nil {
		slog.Error("Error fetching public key", "error", err)
		os.Exit(1)
	}

	pass, err := generateRandomHex(32)
	if err != nil {
		slog.Error("Error generating random key", "error", err)
		os.Exit(1)
	}

	encryptedPass, err := encryptRSA(publicKey, []byte(pass))
	if err != nil {
		slog.Error("Error encrypting the key", "error", err)
		os.Exit(1)
	}

	encryptedData, err := createAndEncryptArchive(paths, pass)
	if err != nil {
		slog.Error("Error creating or encrypting archive", "error", err)
		os.Exit(1)
	}

	err = createSelfExtractingScript(archiveFile, username, encryptedPass, encryptedData)
	if err != nil {
		slog.Error("Error creating self-extracting script", "error", err)
		os.Exit(1)
	}

	slog.Info("Self-extracting archive created", slog.String("archiveFile", archiveFile))
}

func fetchGitHubKey(username string) (*rsa.PublicKey, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	keys, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Split all keys by newline
	keyLines := strings.Split(string(keys), "\n")
	if len(keyLines) == 0 || keyLines[0] == "" {
		return nil, errors.New("no keys found for user")
	}

	for _, keyLine := range keyLines {
		// Trim the key for safety
		keyLine = strings.TrimSpace(keyLine)

		// Skip processing if the line is empty
		if keyLine == "" {
			continue
		}

		// Only attempt to parse keys with the "ssh-rsa" prefix
		if strings.HasPrefix(keyLine, "ssh-rsa") {
			parsedKey, err := parsePublicKey(keyLine)
			if err != nil {
				// Log parsing errors and skip to the next key
				slog.Warn("Failed to parse RSA key", "keyLine", keyLine, "error", err)
				continue
			}

			// Return the first valid RSA public key
			return parsedKey, nil
		}
	}

	// Error if no valid RSA keys are found after filtering
	return nil, errors.New("no valid RSA keys found for user")
}

// parsePublicKey parses an SSH public key and supports RSA, Ed25519, and ECDSA key types.
func parsePublicKey(key string) (*rsa.PublicKey, error) {
	// Parse the authorized key (SSH format)
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		return nil, errors.New("failed to parse SSH public key: " + err.Error())
	}

	// Ensure the key is an RSA key (this should only happen if filtering fails upstream)
	if pubKey.Type() != ssh.KeyAlgoRSA {
		return nil, errors.New("key is not an RSA key")
	}

	// Extract the RSA key from the parsed public key
	rsaKey, err := extractRSAPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	return rsaKey, nil
}

// extractRSAPublicKey extracts an RSA public key from an ssh.PublicKey
func extractRSAPublicKey(pubKey ssh.PublicKey) (*rsa.PublicKey, error) {
	cryptoKey, ok := pubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, errors.New("failed to extract RSA public key: type assertion failed")
	}

	// Convert to *rsa.PublicKey
	rsaKey, ok := cryptoKey.CryptoPublicKey().(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("extracted key is not of type RSA")
	}
	return rsaKey, nil
}

// extractEd25519PublicKey extracts an Ed25519 public key from an ssh.PublicKey
func extractEd25519PublicKey(pubKey ssh.PublicKey) (ed25519.PublicKey, error) {
	cryptoKey, ok := pubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, errors.New("failed to extract Ed25519 public key: type assertion failed")
	}

	// Convert to ed25519.PublicKey
	ed25519Key, ok := cryptoKey.CryptoPublicKey().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("extracted key is not of type Ed25519")
	}
	return ed25519Key, nil
}

// extractECDSAPublicKey extracts an ECDSA public key from an ssh.PublicKey
func extractECDSAPublicKey(pubKey ssh.PublicKey) (*ecdsa.PublicKey, error) {
	cryptoKey, ok := pubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, errors.New("failed to extract ECDSA public key: type assertion failed")
	}

	// Convert to *ecdsa.PublicKey
	ecdsaKey, ok := cryptoKey.CryptoPublicKey().(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("extracted key is not of type ECDSA")
	}
	return ecdsaKey, nil
}

func generateRandomHex(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func encryptRSA(publicKey *rsa.PublicKey, data []byte) (string, error) {
	encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

func createAndEncryptArchive(paths []string, pass string) (string, error) {
	var buffer strings.Builder

	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, path := range paths {
		err := addPathToArchive(tarWriter, path)
		if err != nil {
			return "", err
		}
	}

	err := tarWriter.Close()
	if err != nil {
		return "", err
	}
	err = gzipWriter.Close()
	if err != nil {
		return "", err
	}

	encryptedData, err := encryptAES(buffer.String(), pass)
	if err != nil {
		return "", err
	}

	return encryptedData, nil
}

func addPathToArchive(tw *tar.Writer, path string) error {
	return filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, file)
		if err != nil {
			return err
		}

		header.Name = file
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			fileData, err := os.Open(file)
			if err != nil {
				return err
			}
			defer func(fileData *os.File) {
				err := fileData.Close()
				if err != nil {
					panic(err)
				}
			}(fileData)

			_, err = io.Copy(tw, fileData)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func encryptAES(data, pass string) (string, error) {
	key := []byte(pass)[:32] // Ensure the key is 32 bytes (AES-256)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Pad the data to ensure its length is a multiple of the block size
	paddedData := pkcs7Pad([]byte(data), block.BlockSize())

	// Generate a random nonce (initialization vector, IV)
	nonce := make([]byte, block.BlockSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	// Encrypt the padded data using CBC
	ciphertext := make([]byte, len(paddedData))
	stream := cipher.NewCBCEncrypter(block, nonce)
	stream.CryptBlocks(ciphertext, paddedData)

	// Prepend the nonce to the ciphertext and base64 encode the result
	encoded := base64.StdEncoding.EncodeToString(append(nonce, ciphertext...))
	return encoded, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize) // Calculate the padding size
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func createSelfExtractingScript(filename, username, encryptedPass, encryptedData string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	script := fmt.Sprintf(`#!/usr/bin/env bash
usage() {
  echo "usage: bash $0 identity-file"
  echo "encrypted using: github.com/%s.keys"
  exit 1
}

(( $# >= 1 )) || usage
trap "rm -f /tmp/pass.$$" 0

openssl rsautl -decrypt -inkey "$1" -out /tmp/pass.$$ -in <(echo "%s" | base64 -d)
echo "%s" | base64 -d | openssl enc -aes-256-cbc -d -a -pass file:/tmp/pass.$$ | tar xvz
`, username, encryptedPass, encryptedData)

	_, err = file.WriteString(script)
	if err != nil {
		return err
	}

	return os.Chmod(filename, 0755)
}

func encryptWithKey(key any, data []byte) (string, error) {
	switch k := key.(type) {
	case *rsa.PublicKey:
		// Use RSA encryption
		encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, k, data, nil)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(encryptedBytes), nil

	case ed25519.PublicKey:
		// Ed25519 does not support encryption, raise an error
		return "", errors.New("encryption using Ed25519 keys is not supported")

	case *ecdsa.PublicKey:
		// Implement ECDSA encryption here if necessary
		// Note: ECDSA is generally used for signatures, not encryption.
		return "", errors.New("encryption using ECDSA keys is not supported")

	default:
		return "", errors.New("unsupported key type for encryption")
	}
}
