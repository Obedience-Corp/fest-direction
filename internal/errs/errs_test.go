package errs

import (
	"errors"
	"io/fs"
	"testing"
)

func TestWrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		op      string
		err     error
		wantNil bool
		wantIs  error
		wantMsg string
	}{
		{name: "nil cause stays nil", op: "read fest.yaml", err: nil, wantNil: true},
		{name: "sentinel survives wrapping", op: "hash tree", err: ErrInvalidInput, wantIs: ErrInvalidInput, wantMsg: "hash tree: invalid input"},
		{name: "nested wraps unwrap to the root", op: "outer", err: Wrap("inner", fs.ErrNotExist), wantIs: fs.ErrNotExist, wantMsg: "outer: inner: file does not exist"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Wrap(tc.op, tc.err)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("Wrap(%q, nil) = %v, want nil", tc.op, got)
				}
				return
			}
			if !errors.Is(got, tc.wantIs) {
				t.Fatalf("errors.Is(%v, %v) = false", got, tc.wantIs)
			}
			if got.Error() != tc.wantMsg {
				t.Fatalf("Error() = %q, want %q", got.Error(), tc.wantMsg)
			}
			var opErr *OpError
			if !errors.As(got, &opErr) || opErr.Op != tc.op {
				t.Fatalf("errors.As OpError op = %q, want %q", opErr.Op, tc.op)
			}
		})
	}
}
