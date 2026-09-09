package cmdutil

import (
	"context"
	"fmt"
	"net/http"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// Factory provides shared dependencies injected into command packages.
type Factory struct {
	Config             config.Config
	HttpClient         func() (*http.Client, error)
	Context            func() context.Context
	BrowserOpener      browser.Opener
	RepositoryResolver RepositoryResolver
	// GitConfig runs Git configuration operations for auth identity sync.
	GitConfig func(args ...string) (string, error)
}

// CommandContext returns the active root command context when available.
func (f *Factory) CommandContext() context.Context {
	if f != nil && f.Context != nil {
		if ctx := f.Context(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

// NewAPIClient creates an api.Client using the Factory's HTTP client if
// available, falling back to a default client otherwise.
func (f *Factory) NewAPIClient(token string) (*api.Client, error) {
	if f == nil || f.HttpClient == nil {
		return api.NewClient(token).WithContext(f.CommandContext()), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient).WithContext(f.CommandContext()), nil
}

// NewActionsClient creates an Actions client bound to the active command
// context while preserving any injected HTTP transport.
func (f *Factory) NewActionsClient(token string) (*actions.Client, error) {
	if f == nil || f.HttpClient == nil {
		return actions.NewClient(token).WithContext(f.CommandContext()), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return actions.NewClientWithHTTPClient(token, httpClient).WithContext(f.CommandContext()), nil
}
