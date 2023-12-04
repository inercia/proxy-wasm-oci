package cmd

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/inercia/proxy-wasm-oci/pkg/publisher"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
	"github.com/inercia/proxy-wasm-oci/pkg/utils"
)

const publishDesc = `
Publishes a Proxy-Wasm extension to a registry.

The publish command will look for a "main.wasm" file by default,
package it as a OCI image and pushed to the registry.

Example:

  $ pwo publish main.wasm oci://myregistry.com/myrepo
`

func newPublishCmd(cfg *registry.Configuration, l *zap.Logger, out io.Writer) *cobra.Command {
	log := l.Named("publish")
	r := registry.RegistryParams{}
	metaFilename := ""

	cmd := &cobra.Command{
		Use:     "publish [wasm] [remote]",
		Short:   "publish a Wasm to a registry",
		Aliases: []string{"push"},
		Long:    publishDesc,
		Args:    MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("Creating new registry client")
			registryClient, err := registry.NewClientWithParams(r, settings.RegistryConfigFilename, settings.Debug)
			if err != nil {
				return fmt.Errorf("missing registry client: %w", err)
			}

			wasmFile := args[0]
			remote := args[1]

			// try to guess the metadata file if it has not been provided
			if metaFilename == "" {
				if m := strings.TrimSuffix(wasmFile, ".wasm") + ".yaml"; utils.IsFileExists(m) {
					metaFilename = m
				} else if m := path.Join(path.Dir(wasmFile), "Wasm.yaml"); utils.IsFileExists(m) {
					metaFilename = m
				} else {
					return fmt.Errorf("no metadata file (Wasm.yaml) found for %s", wasmFile)
				}
			}

			client := publisher.NewPush(settings, cfg,
				publisher.WithPushRegistryClient(registryClient),
				publisher.WithPushConfig(cfg),
				registry.WithTLSClientConfig(r.CertFile, r.KeyFile, r.CAFile),
				registry.WithInsecure(r.Insecure),
				registry.WithPlainHTTP(r.PlainHTTP),
				publisher.WithPushOptWriter(out))

			client.Settings = settings

			log.Sugar().Infof("Pushing Wasm %q to %q", wasmFile, remote)
			output, err := client.Run(wasmFile, metaFilename, remote)
			if err != nil {
				return err
			}
			log.Sugar().Infof("Push result: %s", output)
			return nil
		},
	}

	f := cmd.Flags()
	registry.AddRegistryParamsFlags(f, &r)
	f.StringVar(&metaFilename, "metadata", "", "filename of the metadata file (Wasm.yaml) to use")

	return cmd
}
