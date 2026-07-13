package version

import (
	"encoding/json"
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"github.com/spf13/cobra"
)

func NewCmdVersion() *cobra.Command {
	var opts struct {
		JSON bool
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()

			if opts.JSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"ag version %s (commit: %s, built: %s)\n",
				info.Version, info.Commit, info.BuildDate)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output version information as JSON")

	return cmd
}
