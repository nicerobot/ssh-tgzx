package constants

import "fmt"

// Constant is a sentinel error type that supports wrapping.
type Constant string

func (e Constant) Error() string { return string(e) }

// Wrap returns a new error wrapping err with additional context.
func (e Constant) Wrap(err error, args ...any) error {
	msg := string(e)
	if len(args) > 0 {
		msg += ": " + fmt.Sprint(args...)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return fmt.Errorf("%s: %w", msg, e)
}

const (
	ErrMissingArgument Constant = "missing required argument"
	ErrFetchKeys       Constant = "failed to fetch keys"
	ErrParseKey        Constant = "failed to parse key"
	ErrNoValidKeys     Constant = "no valid keys found"
	ErrCreateArchive   Constant = "failed to create archive"
	ErrEncrypt         Constant = "failed to encrypt"
	ErrDecrypt         Constant = "failed to decrypt"
	ErrExtract         Constant = "failed to extract"
	ErrOpenFile        Constant = "failed to open file"
	ErrParseIdentity   Constant = "failed to parse identity"
)
