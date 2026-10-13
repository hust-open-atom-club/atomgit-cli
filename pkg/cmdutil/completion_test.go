package cmdutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterValueCompletion(t *testing.T) {
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().String("state", "open", "")
	RegisterValueCompletion(cmd, "state", []string{"open", "closed"})
	complete, ok := cmd.GetFlagCompletionFunc("state")
	if !ok {
		t.Fatal("completion was not registered")
	}
	matches, directive := complete(cmd, nil, "cl")
	if strings.Join(matches, ",") != "closed" || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("matches = %v, directive = %v", matches, directive)
	}
}

func TestRegisterValueCompletionPanicsOnInvalidDefinition(t *testing.T) {
	for _, tt := range []struct {
		name, want string
		duplicate  bool
	}{
		{"missing flag", "does not exist", false},
		{"duplicate registration", "already registered", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "list"}
			if tt.duplicate {
				cmd.Flags().String("state", "open", "")
				RegisterValueCompletion(cmd, "state", []string{"open"})
			}
			defer func() {
				err, ok := recover().(error)
				if !ok || !strings.Contains(err.Error(), `command "list" flag "state"`) || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("panic = %v", err)
				}
				if errors.Unwrap(err) == nil {
					t.Fatal("registration cause lost")
				}
			}()
			RegisterValueCompletion(cmd, "state", []string{"open"})
		})
	}
}
