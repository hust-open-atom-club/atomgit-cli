package cmdutil

import (
	"net/http"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

type Factory struct {
	Config             config.Config
	HttpClient         func() (*http.Client, error)
	BrowserOpener      browser.Opener
	RepositoryResolver RepositoryResolver
	// GitConfig runs Git configuration operations for auth identity sync.
	GitConfig func(args ...string) (string, error)
}
