package cmdutil

import (
	"net/http"

	"github.com/shinwell/ag-cli/internal/config"
)

type Factory struct {
	Config     config.Config
	HttpClient func() (*http.Client, error)
}
