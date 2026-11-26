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
	description := cmd.Long
	if description == "" {
		description = cmd.Short
	}
	cmd.Long = description + "\n\nDeprecated: " + message
	previous := cmd.PreRunE
	previousRun := cmd.PreRun
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		warning := fmt.Appendf(nil, "Command %q is deprecated, %s\n", cmd.Name(), message)
		// A failed optional warning must not block the read or leave bytes
		// buffered for a later fatal stderr flush.
		if writer, ok := cmd.ErrOrStderr().(interface{ WriteBestEffort([]byte) }); ok {
			writer.WriteBestEffort(warning)
		} else {
			_, _ = cmd.ErrOrStderr().Write(warning)
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
