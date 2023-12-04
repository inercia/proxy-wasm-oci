package downloader

import (
	"fmt"
	"strings"

	"github.com/inercia/proxy-wasm-oci/pkg/config"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

// Pull is the action for pulling the given WASM extension.
type Pull struct {
	CommonPullOptions

	Settings *config.GlobalSettings

	RegistryConfig *registry.Configuration

	DestDir string
}

type PullOpt func(*Pull)

func WithConfig(cfg *registry.Configuration) PullOpt {
	return func(p *Pull) {
		p.RegistryConfig = cfg
	}
}

func WithDestDir(d string) PullOpt {
	return func(p *Pull) {
		p.DestDir = d
	}
}

func WithVersion(v string) PullOpt {
	return func(p *Pull) {
		p.Version = v
	}
}

// NewPull creates a new pull, with configuration options.
func NewPull(settings *config.GlobalSettings, cfg *registry.Configuration, opts ...any) *Pull {
	p := &Pull{
		Settings:       settings,
		RegistryConfig: cfg,
	}
	for _, fn := range opts {
		switch r := fn.(type) {
		case PullOpt:
			r(p)
		case registry.RegistryParamsOpt:
			r(&p.RegistryParams)
		default:
			panic("unknown type passed to NewPull")
		}
	}

	return p
}

// SetRegistryClient sets the registry client on the pull configuration object.
func (p *Pull) SetRegistryClient(client *registry.Client) {
	p.RegistryConfig.RegistryClient = client
}

// Run performans a 'pull' of the given WASM extension.
func (p *Pull) Run(remote string) (string, error) {
	var out strings.Builder

	if !registry.IsOCI(remote) {
		return out.String(), fmt.Errorf("%q is not a valid OCI reference", remote)
	}

	downloader := WASMDownloader{
		Out:     &out,
		Verify:  VerifyNever,
		Getters: All(p.Settings),
		Options: []Option{
			WithBasicAuth(p.Username, p.Password),
			WithPassCredentialsAll(p.PassCredentialsAll),
			WithTLSClientConfig(p.CertFile, p.KeyFile, p.CAFile),
			WithInsecureSkipVerifyTLS(p.Insecure),
			WithPlainHTTP(p.PlainHTTP),
			WithRegistryClient(p.RegistryConfig.RegistryClient),
		},
		RegistryClient: p.RegistryConfig.RegistryClient,
	}

	if p.Verify {
		downloader.Verify = VerifyAlways
	}

	saved, _, err := downloader.DownloadTo(remote, p.Version, p.DestDir)
	if err != nil {
		return out.String(), err
	}

	if p.Verify {
		// TODO
	}

	return saved, nil
}
