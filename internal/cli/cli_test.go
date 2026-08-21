package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func run(ctx context.Context, args ...string) (string, error) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return out.String(), err
}

func TestRootCommand(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		args    []string
		wantErr bool
		wantIs  error
		wantOut string
	}{
		{name: "cancelled context is refused before any verb runs", ctx: cancelled, args: []string{"version"}, wantErr: true, wantIs: context.Canceled},
		{name: "unknown verb is an error", ctx: context.Background(), args: []string{"nope"}, wantErr: true},
		{name: "version prints build metadata", ctx: context.Background(), args: []string{"--no-color", "version"}, wantOut: "dev"},
		{name: "help lists verbs", ctx: context.Background(), args: []string{"--help"}, wantOut: "version"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := run(tc.ctx, tc.args...)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v (output: %q)", err, tc.wantErr, out)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want errors.Is %v", err, tc.wantIs)
			}
			if tc.wantOut != "" && !strings.Contains(out, tc.wantOut) {
				t.Fatalf("output %q missing %q", out, tc.wantOut)
			}
		})
	}
}
