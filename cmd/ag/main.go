package main

import (
	"os"

	"gitcode.com/openeuler/ag-cli/internal/agcmd"
)

func main() {
	code := agcmd.Main()
	os.Exit(code)
}
