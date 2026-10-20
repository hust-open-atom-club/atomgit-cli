package license

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
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
			if f.HttpClient != nil {
				injected, err := f.HttpClient()
				if err != nil {
					return fmt.Errorf("prepare license check: %w: %w", api.ErrRequestPreparation, err)
				}
				if injected == nil {
					return fmt.Errorf("prepare license check: %w: HTTP client is nil", api.ErrRequestPreparation)
				}
				configuredClient := *injected
				client = &configuredClient
				if client.Timeout == 0 {
					client.Timeout = 30 * time.Second
				}
			}
			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, fullURL, nil)
			if err != nil {
				return fmt.Errorf("prepare license check: %w: %w", api.ErrRequestPreparation, err)
			}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to check license: %w", &api.RequestExecutionError{Err: err})
			}
			defer resp.Body.Close() //nolint:errcheck // Status determines the primary error; the response body is best-effort diagnostic text and Close only releases response resources.

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("license check failed: %w", api.NewHTTPError(resp))
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", &api.ResponseReadError{Err: err})
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
