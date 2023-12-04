package publisher

import (
	"fmt"
	"io"
	"net/url"

	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

// WASMUploader handles uploading a Proxy-WASM extension.
type WASMUploader struct {
	// Out is the location to write warning and info messages.
	Out io.Writer
	// Pusher collection for the operation
	Pushers Providers
	// Options provide parameters to be passed along to the Pusher being initialized.
	Options []Option
	// RegistryClient is a client for interacting with registries.
	RegistryClient *registry.Client
}

func NewWASMUploader(out io.Writer, pushers Providers, opts ...Option) WASMUploader {
	c := WASMUploader{
		Out:     out,
		Pushers: pushers,
		Options: opts,
	}
	return c
}

// UploadTo uploads a chart. Depending on the settings, it may also upload a provenance file.
func (c *WASMUploader) UploadTo(wasmExe, metadataFile, remote string) error {
	remoteURL, err := url.Parse(remote)
	if err != nil {
		return fmt.Errorf("invalid chart URL format: %s", remote)
	}

	if remoteURL.Scheme == "" {
		return fmt.Errorf("scheme prefix missing from remote (e.g. \"%s://\")", registry.OCIScheme)
	}

	p, err := c.Pushers.ByScheme(remoteURL.Scheme)
	if err != nil {
		return err
	}

	return p.Push(wasmExe, metadataFile, remoteURL.String(), c.Options...)
}
