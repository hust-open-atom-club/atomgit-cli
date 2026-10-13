package main

import (
	"os"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/agcmd"
)

func main() {
	code := agcmd.Main()
	os.Exit(code)
}
