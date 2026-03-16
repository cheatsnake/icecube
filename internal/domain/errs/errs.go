package errs

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// ExtractErrorMessage extracts the most meaningful message from an error (including joined errors)
func ExtractErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errs := extractErrors(err)
	// Return the last error message (most specific one in the chain)
	if len(errs) > 0 {
		return errs[len(errs)-1].Error()
	}

	return err.Error()
}

// extractErrors extracts all errors from a joined error chain
func extractErrors(err error) []error {
	var joinErr interface {
		Unwrap() []error
	}

	if errors.As(err, &joinErr) {
		return joinErr.Unwrap()
	}
	return []error{err}
}
