package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/mangen"
	rootcmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fail(err)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("generate-manpages", flag.ContinueOnError)
	flags.SetOutput(out)
	output := flags.String("output", "dist/man/man1", "manual output directory (never a system directory by default)")
	version := flags.String("version", "", "explicit build version for the header")
	date := flags.String("date", "", "explicit header date in YYYY-MM-DD format")
	check := flags.Bool("check", false, "check every existing manual against the command tree")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	root, err := rootcmd.NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		return err
	}
	pages, err := mangen.Generate(root, mangen.Header{Version: *version, Date: *date})
	if err == nil {
		err = mangen.Write(*output, pages, *check)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%d manpages verified in %s\n", len(pages), *output)
	return err
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "generate-manpages:", err)
	os.Exit(1)
}
