package schema

import (
	"encoding/json"
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"github.com/spf13/cobra"
)

func NewCmdSchema() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schema [<command> ...]",
		Short:   "Describe public commands as versioned JSON",
		Long:    "List public commands or describe an exact command path without executing it. Paths may include the ag prefix; use 'ag schema ag' for root command details. Uses static command metadata only; no login or network is required. This is a versioned command description format, not JSON Schema. Undescribed behavior must not be inferred.",
		Example: "  ag schema\n  ag schema ag\n  ag schema pr create\n  ag schema ag pr create\n  ag schema api\n  ag schema pr comment create",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := commandschema.Describe(cmd.Root(), args)
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(document); err != nil {
				return fmt.Errorf("write command description: %w", err)
			}
			return nil
		},
	}
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: -1, Description: "Exact public command path components, optionally prefixed with ag; use ag alone for root details or omit to list commands. Built-in aliases are accepted; local aliases are excluded."},
		Output:      "json", Effects: "none",
	})
	return cmd
}
