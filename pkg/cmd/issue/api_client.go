package issue

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func newAPIClient(f *cmdutil.Factory, token string) (*api.Client, error) {
	if f.HttpClient == nil {
		return api.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), nil
}
