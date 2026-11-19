package user

import (
	"fmt"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type followerJSON struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	AvatarURL string `json:"avatarUrl"`
}

func newCmdUserFollowers(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use: "followers <username>", Short: "List followers of a user",
		Long:    "List followers of an explicitly specified user. Uses authentication when configured; otherwise reads publicly available followers anonymously. The limit applies across all pages.",
		Example: "  ag-cli user followers alice\n  ag-cli user followers alice --limit 100\n  ag-cli user followers alice --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimSpace(args[0])
			if !validLogin(username) || username == "." || username == ".." || strings.ContainsRune(username, '\\') || strings.ContainsFunc(username, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
				return fmt.Errorf("invalid username %q: expected a user name without whitespace, control characters, or path/query separators", args[0])
			}
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}
			client, err := f.OptionalAPIClient()
			if err != nil {
				return err
			}
			followers, err := api.ListUserFollowers(client.WithContext(cmd.Context()), username, limit)
			if err != nil {
				return fmt.Errorf("failed to list followers for user %q: %w", username, err)
			}
			if asJSON {
				result := make([]followerJSON, 0, len(followers))
				for _, follower := range followers {
					result = append(result, followerJSON{ID: follower.ID, Login: follower.Login, Name: follower.Name, Type: follower.Type, URL: follower.HTMLURL, AvatarURL: follower.AvatarURL})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			if len(followers) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No followers found for user %q.\n", username)
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "ID\tLOGIN\tNAME\tTYPE\tURL"); err != nil {
				return err
			}
			for _, follower := range followers {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n",
					cmdutil.EscapeTSVField(follower.ID), cmdutil.EscapeTSVField(follower.Login), cmdutil.EscapeTSVField(follower.Name), cmdutil.EscapeTSVField(follower.Type), cmdutil.EscapeTSVField(follower.HTMLURL)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum total number of followers to list")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output followers as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Explicit username; Git repository context is not used."},
		Output:      "text,json", Effects: "read",
		Notes: []string{"Public reading supports anonymous access when no credentials are configured; credential errors do not fall back to anonymous access.", "--limit must be positive and limits the total across pages. JSON preserves string IDs and the fixed id, login, name, type, url, avatarUrl fields."},
	})
	return cmd
}
