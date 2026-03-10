package cmdutil

import (
	"net/http"

<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/internal/config"
=======
	"atomgit.com/openeuler/ag-cli/internal/config"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
)

type Factory struct {
	Config     config.Config
	HttpClient func() (*http.Client, error)
}
