// Package errs is the project error framework: sentinel errors plus an
// operation-wrapping helper. Production code wraps with Wrap instead of
// fmt.Errorf so every error names the operation that failed and unwraps
// cleanly through errors.Is and errors.As. golangci's forbidigo rule enforces
// the fmt.Errorf ban outside tests.
package errs

import "errors"

var (
	// ErrAlreadyPrinted signals that the user-facing message was already
	// written; main exits non-zero without printing again.
	ErrAlreadyPrinted = errors.New("error already reported")

	// ErrNotImplemented marks a verb or path that a later phase delivers.
	ErrNotImplemented = errors.New("not implemented")

	// ErrInvalidInput marks caller-supplied arguments the command refuses.
	ErrInvalidInput = errors.New("invalid input")
)

// OpError records the operation that failed and its cause. Op carries any
// dynamic detail (paths, field names) so the cause can stay a sentinel.
type OpError struct {
	Op  string
	Err error
}

// Error renders "op: cause".
func (e *OpError) Error() string { return e.Op + ": " + e.Err.Error() }

// Unwrap exposes the cause to errors.Is and errors.As.
func (e *OpError) Unwrap() error { return e.Err }

// Wrap annotates err with op. A nil err returns nil so call sites can wrap
// unconditionally on return.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Err: err}
}
