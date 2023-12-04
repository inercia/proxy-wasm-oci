package cmd

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	reg "oras.land/oras-go/pkg/registry"

	"github.com/inercia/proxy-wasm-oci/pkg/downloader"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
	"github.com/inercia/proxy-wasm-oci/pkg/server"
)

const downloadDesc = `
Retrieve a Proxy-Wasm extension from an OCI registry and save it locally.

This is useful for fetching extensions to inspect, modify, or repackage.

Example:

  $ pwo download --dest /tmp oci://myregistry.com/myrepo:1.0.0
`

func newDownloadCmd(cfg *registry.Configuration, l *zap.Logger, out io.Writer) *cobra.Command {
	log := l.Named("download")
	r := downloader.CommonPullOptions{}
	destDir := ""

	cmd := &cobra.Command{
		Use:     "download [remote]",
		Short:   "download a Proxy-Wasm extensions from an OCI registry into a local directory",
		Aliases: []string{"fetch", "pull"},
		Long:    downloadDesc,
		Args:    MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("Cretaing new registry client")
			registryClient, err := registry.NewClientWithParams(r.RegistryParams, settings.RegistryConfigFilename, settings.Debug)
			if err != nil {
				return fmt.Errorf("when creating registry client: %w", err)
			}

			ref := args[0]
			if !registry.IsOCI(ref) {
				return fmt.Errorf("invalid OCI reference: %s", ref)
			}

			version, err := getVersionFromRef(ref)
			if err != nil {
				return err
			}

			puller := downloader.NewPull(settings, cfg,
				registry.WithTLSClientConfig(r.CertFile, r.KeyFile, r.CAFile),
				registry.WithInsecure(r.Insecure),
				registry.WithPlainHTTP(r.PlainHTTP),
				downloader.WithDestDir(destDir),
				downloader.WithVersion(version),
			)
			puller.SetRegistryClient(registryClient)

			log.Sugar().Infof("Downloading %s", ref)
			output, err := puller.Run(ref)
			if err != nil {
				return err
			}
			log.Sugar().Infof("File downloaded to %s", output)
			log.Sugar().Infof("You could GET this from the server at:")
			log.Sugar().Infof("  %s?%s", server.PathWASMDownload, getRefAsQueryParameters(ref))

			return nil
		},
	}

	f := cmd.Flags()
	downloader.AddDownloadFlags(f, &r)
	registry.AddRegistryParamsFlags(f, &r.RegistryParams)
	f.StringVarP(&destDir, "destination", "d", ".", "location to write the extension.")

	return cmd
}

func getVersionFromRef(ref string) (string, error) {
	refNoScheme := strings.TrimPrefix(ref, fmt.Sprintf("%s://", registry.OCIScheme))
	parsedReference, err := reg.ParseReference(refNoScheme)
	if err != nil {
		return "", err
	}

	if _, err = semver.NewVersion(parsedReference.Reference); parsedReference.Reference != "" && err == nil {
		return parsedReference.Reference, nil
	}

	return ">0.0.0-0", nil
}

func getRefAsQueryParameters(ref string) string {
	return fmt.Sprintf("ref=%s", url.QueryEscape(ref))
}
