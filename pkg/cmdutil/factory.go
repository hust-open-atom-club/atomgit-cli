package cmdutil

import (
	"net/http"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

type Factory struct {
	Config             config.Config
	HttpClient         func() (*http.Client, error)
	RepositoryResolver RepositoryResolver
}
