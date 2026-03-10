package license

import (
	"gitcode.com/openeuler/ag-cli/pkg/cmdutil"
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
