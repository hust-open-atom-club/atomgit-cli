package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// RegisterValueCompletion registers a static candidate completion for a flag's
// value from a fixed candidate set, filtering by the prefix already typed.
// File completion is always disabled so an unmatched prefix falls back to
// nothing instead of a directory listing. Use it only for enums that need no
// network or configuration access, so completion stays correct without
// authentication.
// Registration errors indicate an invalid command definition and panic.
func RegisterValueCompletion(cmd *cobra.Command, flag string, candidates []string) {
	err := cmd.RegisterFlagCompletionFunc(flag, func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		matches := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			if strings.HasPrefix(candidate, toComplete) {
				matches = append(matches, candidate)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		panic(fmt.Errorf("register value completion for command %q flag %q: %w", cmd.Name(), flag, err))
	}
}
