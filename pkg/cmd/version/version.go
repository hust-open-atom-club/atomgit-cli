package version

import (
	"encoding/json"
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"github.com/spf13/cobra"
)

func Text() string {
	return formatText(version.Get())
}

func formatText(info version.Info) string {
	details := make([]string, 0, 2)
	if commit := knownMetadata(info.Commit); commit != "" {
		details = append(details, "commit: "+commit)
	}
	if buildDate := knownMetadata(info.BuildDate); buildDate != "" {
		details = append(details, "built: "+buildDate)
	}
	if len(details) == 0 {
		return fmt.Sprintf("ag version %s\n", info.Version)
	}
	return fmt.Sprintf("ag version %s (%s)\n", info.Version, strings.Join(details, ", "))
}

func knownMetadata(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	return value
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
