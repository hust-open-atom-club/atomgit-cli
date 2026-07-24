package version

import (
	"encoding/json"
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"github.com/spf13/cobra"
)

func Text() string {
	info := version.Get()
	return fmt.Sprintf("ag version %s (commit: %s, built: %s, self-update: %t, source: %s)\n",
		info.Version, info.Commit, info.BuildDate, info.SelfUpdate, info.Source)
}

func NewCmdVersion() *cobra.Command {
	var opts struct {
		JSON bool
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.JSON {
				info := version.Get()
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			_, err := fmt.Fprint(cmd.OutOrStdout(), Text())
			return err
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output version information as JSON")

	return cmd
}
