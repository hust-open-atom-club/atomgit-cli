package main

import (
	"os"

	"github.com/shinwell/ag-cli/internal/agcmd"
)

func main() {
	code := agcmd.Main()
	os.Exit(code)
}
