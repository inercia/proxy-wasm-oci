package server

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"go.uber.org/zap"
	reg "oras.land/oras-go/pkg/registry"

	"github.com/inercia/proxy-wasm-oci/pkg/downloader"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

func DownloadWASMExtension(log *zap.Logger, server *Server, ref string, dir string, r registry.RegistryParams) (string, error) {
	raw, err, _ := server.downloads.Do(ref, func() (interface{}, error) {
		log.Info("Creating new registry client")
		registryClient, err := registry.NewClientWithParams(r, server.settings.RegistryConfigFilename, server.settings.Debug)
		if err != nil {
			return nil, fmt.Errorf("when creating registry client: %w", err)
		}

		if !registry.IsOCI(ref) {
			return nil, fmt.Errorf("invalid OCI reference: %s", ref)
		}

		version := ">0.0.0-0"
		refNoScheme := strings.TrimPrefix(ref, fmt.Sprintf("%s://", registry.OCIScheme))
		parsedReference, err := reg.ParseReference(refNoScheme)
		if err != nil {
			return "", err
		}
		if _, err = semver.NewVersion(parsedReference.Reference); parsedReference.Reference != "" && err == nil {
			log.Sugar().Infof("Downloading version %s", parsedReference.Reference)
			version = parsedReference.Reference
		}

		puller := downloader.NewPull(server.settings, server.registryConfig,
			registry.WithTLSClientConfig(r.CertFile, r.KeyFile, r.CAFile),
			registry.WithInsecure(r.Insecure),
			registry.WithPlainHTTP(r.PlainHTTP),
			downloader.WithDestDir(dir),
			downloader.WithVersion(version),
		)
		puller.SetRegistryClient(registryClient)

		log.Sugar().Infof("Downloading %s", ref)
		savedFilename, err := puller.Run(ref)
		if err != nil {
			return "", err
		}

		return savedFilename, nil
	})

	return raw.(string), err
}
