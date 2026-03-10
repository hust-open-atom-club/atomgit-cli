package license

import (
<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
	"github.com/spf13/cobra"
)

func NewCmdLicense(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "License compliance checking",
		Long:  `Check license compliance using openEuler compliance service.`,
	}

	cmd.AddCommand(newCmdCheck(f))

	return cmd
}
