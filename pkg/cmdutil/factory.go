package cmdutil

import (
	"fmt"
	"net/http"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// Factory provides shared dependencies injected into command packages.
type Factory struct {
	Config             config.Config
	HttpClient         func() (*http.Client, error)
	BrowserOpener      browser.Opener
	RepositoryResolver RepositoryResolver
	// GitConfig runs Git configuration operations for auth identity sync.
	GitConfig func(args ...string) (string, error)
}

// NewAPIClient creates an api.Client using the Factory's HTTP client if
// available, falling back to a default client otherwise.
func (f *Factory) NewAPIClient(token string) (*api.Client, error) {
	if f.HttpClient == nil {
		return api.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), nil
}
