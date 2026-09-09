package license

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdCheck(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "check <license>",
		Short: "Check license compliance",
		Long:  `Check if a license is compliant using openEuler compliance service.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			license := args[0]

			// Build URL
			baseURL := "https://compliance.openeuler.org/check"
			params := url.Values{}
			params.Add("license", license)
			fullURL := baseURL + "?" + params.Encode()

			// Make HTTP GET request with timeout
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Get(fullURL)
			if err != nil {
				return fmt.Errorf("failed to check license: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				return fmt.Errorf("license check failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			// Check if body is empty
			if len(body) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "未知")
				return nil
			}

			// Print API response
			fmt.Fprintln(cmd.OutOrStdout(), string(body))
			return nil
		},
	}
}
