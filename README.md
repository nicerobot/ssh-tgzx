# ssh-tgzx

Create and extract age-encrypted tar.gz archives secured with SSH keys from GitHub.

## Install

```bash
go install github.com/nicerobot/ssh-tgzx/cmd/ssh-tgzx@latest
```

Or download a pre-built binary from [Releases](https://github.com/nicerobot/ssh-tgzx/releases).

## Usage

### Create an encrypted archive

Archive files for a GitHub user using their public SSH keys:

```bash
ssh-tgzx create <github-username> <archive-file> <paths...>
```

Example:

```bash
ssh-tgzx create nicerobot private.age secret-folder/ credentials.txt
```

### Extract an archive

Decrypt and extract using your SSH private key:

```bash
ssh-tgzx extract <archive-file> <identity-file>
```

Example:

```bash
ssh-tgzx extract private.age ~/.ssh/id_ed25519
```

### List archive contents

List files without extracting:

```bash
ssh-tgzx list <archive-file> <identity-file>
```

Example:

```bash
ssh-tgzx list private.age ~/.ssh/id_ed25519
```

## How it works

1. **Create**: Fetches the recipient's SSH public keys from `github.com/<username>.keys`, creates a tar.gz of the specified files, and encrypts it using [age](https://age-encryption.org/) with the SSH public keys as recipients.

2. **Extract/List**: Reads the SSH private key, decrypts the age-encrypted archive, and extracts or lists the tar.gz contents.

## Supported key types

- **Ed25519** (recommended)
- **RSA**

ECDSA keys are not supported by the age encryption library and will be skipped.

## macOS quarantine

macOS XProtect may flag downloaded binaries. To allow:

1. Open **System Settings > Privacy & Security**
2. Look for the blocked file notice and click **Allow Anyway**

Or remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine ssh-tgzx
```
