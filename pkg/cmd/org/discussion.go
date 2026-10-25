package org

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussion(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "discussion",
		Short:        "Manage organization discussions",
		Long:         "Create, edit, and delete organization discussions and their comments. An explicit organization path is required for every operation.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newCmdOrgDiscussionCreate(f))
	cmd.AddCommand(newCmdOrgDiscussionEdit(f))
	cmd.AddCommand(newCmdOrgDiscussionDelete(f))
	cmd.AddCommand(newCmdOrgDiscussionComment(f))
	commandschema.Annotate(cmd, commandschema.Metadata{Output: "text", Effects: "none"})
	return cmd
}

func parseDiscussionOrganization(value string) (string, error) {
	if !validDiscussionPathID(value) {
		return "", fmt.Errorf("invalid organization: %q", value)
	}
	return value, nil
}

func orgDiscussionNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 || strconv.Itoa(number) != value {
		return 0, fmt.Errorf("invalid discussion number %q: must be a positive integer", value)
	}
	return number, nil
}

func validDiscussionPathID(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r)) {
			return false
		}
	}
	return true
}
