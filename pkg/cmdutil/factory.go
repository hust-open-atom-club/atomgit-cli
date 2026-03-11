package cmdutil

import (
	"net/http"

	"atomgit.com/openeuler/ag-cli/internal/config"
)

type Factory struct {
	Config     config.Config
	HttpClient func() (*http.Client, error)
}
