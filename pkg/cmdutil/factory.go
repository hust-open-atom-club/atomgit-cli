package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

// AuthenticatedAPIClient builds a v5 API client using stored credentials.
// Missing credentials and credential-store failures become AuthenticationError.
func (f *Factory) AuthenticatedAPIClient() (*api.Client, error) {
	token, err := f.requiredToken()
	if err != nil {
		return nil, err
	}
	return f.NewAPIClient(token)
}

// OptionalAPIClient builds a v5 API client for publicly readable endpoints.
// A missing login becomes an anonymous request; credential-store failures do not.
func (f *Factory) OptionalAPIClient() (*api.Client, error) {
	token, err := f.optionalToken()
	if err != nil {
		return nil, err
	}
	return f.NewAPIClient(token)
}

// AuthenticatedActionsClient builds a v8 Actions client using stored credentials.
func (f *Factory) AuthenticatedActionsClient() (*actions.Client, error) {
	token, err := f.requiredToken()
	if err != nil {
		return nil, err
	}
	return f.NewActionsClient(token)
}

func (f *Factory) requiredToken() (string, error) {
	if f == nil || f.Config == nil {
		return "", fmt.Errorf("configuration is unavailable")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return "", AuthenticationError(err)
	}
	if strings.TrimSpace(token) == "" {
		return "", AuthenticationError(config.ErrNotAuthenticated)
	}
	return token, nil
}

func (f *Factory) optionalToken() (string, error) {
	if f == nil || f.Config == nil {
		return "", fmt.Errorf("configuration is unavailable")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		if errors.Is(err, config.ErrNotAuthenticated) {
			return "", nil
		}
		return "", AuthenticationError(err)
	}
	return token, nil
}
