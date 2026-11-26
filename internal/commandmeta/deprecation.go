// Package commandmeta holds command metadata shared by execution and documentation.
package commandmeta

import (
	"fmt"

	"github.com/spf13/cobra"
)

const deprecationKey = "ag-cli.deprecation"

// Deprecate keeps a compatibility command out of normal help and completion,
// like Cobra's Deprecated field, but writes its warning to stderr. Cobra's
// automatic deprecation notice goes to stdout and corrupts JSON and file data.
func Deprecate(cmd *cobra.Command, message string) {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	cmd.Annotations[deprecationKey] = message
	cmd.Hidden = true
	previous := cmd.PreRunE
	previousRun := cmd.PreRun
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Command %q is deprecated, %s\n", cmd.Name(), message); err != nil {
			return fmt.Errorf("write deprecation warning: %w", err)
		}
		if previous != nil {
			return previous(cmd, args)
		}
		if previousRun != nil {
			previousRun(cmd, args)
		}
		return nil
	}
}

// Deprecation returns deprecation metadata for either warning mechanism.
func Deprecation(cmd *cobra.Command) string {
	if message := cmd.Annotations[deprecationKey]; message != "" {
		return message
	}
	return cmd.Deprecated
}

// Hidden reports genuinely private commands, excluding compatibility commands
// hidden only from normal Cobra help and completion by Deprecate.
func Hidden(cmd *cobra.Command) bool {
	return cmd.Hidden && cmd.Annotations[deprecationKey] == ""
}
