package cmd

import (
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/inercia/proxy-wasm-oci/pkg/config"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

var globalUsage = `The Proxy-WASM for OCI (PWO) utility

Common actions for PWO:

- pwo download:      download a Proxy-Wasm to your local directory to view
- pwo publish:       upload the Proxy-Wasm to the regisrty
- pwo serve:         serve the Proxy-Wasm from the registry, acting as a bridge between the Envoy and the registry.

By default, the default directories depend on the Operating System. The defaults are listed below:

| Operating System | Cache Path                | Configuration Path             | Data Path               |
|------------------|---------------------------|--------------------------------|-------------------------|
| Linux            | $HOME/.cache/pwo          | $HOME/.config/pwo              | $HOME/.local/share/pwo  |
| macOS            | $HOME/Library/Caches/pwo  | $HOME/Library/Preferences/pwo  | $HOME/Library/pwo       |
| Windows          | %TEMP%\pwo                | %APPDATA%\pwo                  | %APPDATA%\pwo           |
`

var rootCmd = &cobra.Command{
	Use:          "pwo",
	Short:        "Proxy-WASM utility.",
	Long:         globalUsage,
	SilenceUsage: true,
}

var settings = config.New()

func newRootCmd(cfg *registry.Configuration, l *zap.Logger, out io.Writer, args []string) (*cobra.Command, error) {
	flags := rootCmd.PersistentFlags()

	// We can safely ignore any errors that flags.Parse encounters since
	// those errors will be caught later during the call to cmd.Execution.
	// This call is required to gather configuration information prior to
	// execution.
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Parse(args)

	registryClient, err := registry.NewDefaultRegistryClient(false)
	if err != nil {
		return nil, err
	}
	cfg.RegistryClient = registryClient

	return rootCmd, nil
}

func init() {
	log.SetFlags(log.Lshortfile)
}

func Execute() {
	out := io.Writer(os.Stdout)
	cfg := new(registry.Configuration)

	log, err := zap.NewDevelopment()
	if err != nil {
		warning("%+v", err)
		os.Exit(1)
	}

	rootCmd, err := newRootCmd(cfg, log, out, os.Args[1:])
	if err != nil {
		warning("%+v", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(newPublishCmd(cfg, log, out))
	rootCmd.AddCommand(newDownloadCmd(cfg, log, out))
	rootCmd.AddCommand(newServeCmd(cfg, log, out))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
