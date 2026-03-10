package cmdutil

import (
	"net/http"

	"gitcode.com/openeuler/ag-cli/internal/config"
)

type Factory struct {
	Config     config.Config
	HttpClient func() (*http.Client, error)
}
