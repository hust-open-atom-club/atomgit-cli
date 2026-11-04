package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// Factory provides shared dependencies injected into command packages.
type Factory struct {
	// BaseURL overrides the default AtomGit server origin for this process only.
	BaseURL            string
	BaseURLOverrideSet bool
	Config             config.Config
	TokenOverride      string
	TokenOverrideSet   bool
	HttpClient         func() (*http.Client, error)
	Context            func() context.Context
	BrowserOpener      browser.Opener
	RepositoryResolver RepositoryResolver
	// ErrorWriter receives transient API retry notices. Root commands install
	// their sanitized stderr writer here; nil keeps library-style clients quiet.
	ErrorWriter io.Writer
	// GitConfig runs Git configuration operations for auth identity sync.
	GitConfig func(args ...string) (string, error)
}

// ApplyTokenOverride installs the command-line token as an in-memory view of
// the loaded configuration. It never writes to the credential store.
func (f *Factory) ApplyTokenOverride() {
	if f == nil || f.Config == nil || !f.HasTokenOverride() {
		return
	}
	f.Config = config.WithTokenOverride(f.Config, &f.TokenOverride, f.resolveTokenUser)
}

// HasTokenOverride reports whether a token was explicitly selected for this
// command. The Changed state keeps an explicit empty value distinct from an
// omitted flag; the non-empty fallback supports factories constructed by
// callers other than Cobra.
func (f *Factory) HasTokenOverride() bool {
	return f != nil && (f.TokenOverrideSet || strings.TrimSpace(f.TokenOverride) != "")
}

// HasBaseURLOverride reports whether the API origin was explicitly selected.
// A non-default value also counts for factories constructed outside Cobra.
func (f *Factory) HasBaseURLOverride() bool {
	if f == nil {
		return false
	}
	if f.BaseURLOverrideSet {
		return true
	}
	origin, err := NormalizeBaseURL(f.BaseURL)
	return err == nil && !strings.EqualFold(origin, api.BaseURL)
}

// TokenForCommand returns an explicitly supplied command-line token when one
// is available, otherwise it returns fallback. The boolean reports whether the
// command-line token was selected, allowing diagnostic commands to keep local
// credential inspection separate from credentials used for online probes.
func (f *Factory) TokenForCommand(fallback string) (string, bool) {
	if f != nil && f.HasTokenOverride() {
		return strings.TrimSpace(f.TokenOverride), true
	}
	return fallback, false
}

func (f *Factory) resolveTokenUser(token string) (string, error) {
	client, err := f.NewAPIClient(token)
	if err != nil {
		return "", fmt.Errorf("resolve user for command-line token: %w", err)
	}
	user, err := client.CurrentUserLogin()
	if err != nil {
		return "", fmt.Errorf("resolve user for command-line token: %w", err)
	}
	return user, nil
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
	baseURL := ""
	if f != nil {
		baseURL = f.BaseURL
	}
	if baseURL != "" {
		var err error
		baseURL, err = APIBaseURL(baseURL, api.APIVersion)
		if err != nil {
			return nil, err
		}
	}
	if f == nil || f.HttpClient == nil {
		if baseURL != "" {
			return api.NewClientWithBaseURL(token, baseURL, nil).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
		}
		return api.NewClient(token).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	if baseURL != "" {
		return api.NewClientWithBaseURL(token, baseURL, httpClient).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
	}
	return api.NewClientWithHTTPClient(token, httpClient).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
}

// NewActionsClient creates an Actions client bound to the active command
// context while preserving any injected HTTP transport.
func (f *Factory) NewActionsClient(token string) (*actions.Client, error) {
	baseURL := ""
	if f != nil {
		baseURL = f.BaseURL
	}
	if baseURL != "" {
		var err error
		baseURL, err = APIBaseURL(baseURL, actions.APIVersion)
		if err != nil {
			return nil, err
		}
	}
	if f == nil || f.HttpClient == nil {
		if baseURL != "" {
			return actions.NewClientWithBaseURL(token, baseURL, nil).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
		}
		return actions.NewClient(token).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	if baseURL != "" {
		return actions.NewClientWithBaseURL(token, baseURL, httpClient).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
	}
	return actions.NewClientWithHTTPClient(token, httpClient).WithContext(f.CommandContext()).WithRetryWriter(factoryErrorWriter(f)), nil
}

// NormalizeBaseURL validates and normalizes a server origin supplied on the
// command line. API version paths are added separately by the client factory.
func NormalizeBaseURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base URL: expected an HTTP(S) server origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid base URL: scheme must be http or https")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return "", fmt.Errorf("invalid base URL: http is only allowed for loopback hosts")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("invalid base URL: credentials, query, fragment, and path are not allowed")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// APIBaseURL appends an API version path to a validated server origin.
func APIBaseURL(serverOrigin, apiVersion string) (string, error) {
	origin, err := NormalizeBaseURL(serverOrigin)
	if err != nil {
		return "", err
	}
	return origin + "/" + strings.TrimLeft(apiVersion, "/"), nil
}

func factoryErrorWriter(f *Factory) io.Writer {
	if f == nil {
		return nil
	}
	return f.ErrorWriter
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
