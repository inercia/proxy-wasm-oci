package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

const registryLogoutDesc = `
Remove credentials stored for a remote registry.
`

func newRegistryLogoutCmd(cfg *registry.Configuration, out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "logout [host]",
		Short: "logout from a registry",
		Long:  registryLogoutDesc,
		Args:  MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := args[0]
			return registry.NewRegistryLogout(cfg).Run(out, hostname)
		},
	}
}
