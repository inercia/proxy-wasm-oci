package downloader

import (
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

// CommonPullOptions captures common options used for controlling pulls
type CommonPullOptions struct {
	registry.RegistryParams

	Password           string // --password
	PassCredentialsAll bool   // --pass-credentials
	Username           string // --username
	Verify             bool   // --verify
	Version            string // --version

	// registryClient provides a registry client but is not added with
	// options from a flag
	registryClient *registry.Client
}
