package main

import (
	"os"

	"atomgit.com/openeuler/ag-cli/internal/agcmd"
)

func main() {
	code := agcmd.Main()
	os.Exit(code)
}
