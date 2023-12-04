package publisher

import (
	"fmt"
	"io"
	"strings"

	"github.com/inercia/proxy-wasm-oci/pkg/config"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

// Push is the action for uploading a chart.
//
// It provides the implementation of 'helm push'.
type Push struct {
	Settings *config.GlobalSettings
	cfg      *registry.Configuration
	registry.RegistryParams
	out io.Writer
}

// PushOpt is a type of function that sets options for a push action.
type PushOpt func(*Push)

// WithPushConfig sets the cfg field on the push configuration object.
func WithPushConfig(cfg *registry.Configuration) PushOpt {
	return func(p *Push) {
		p.cfg = cfg
	}
}

// WithOptWriter sets the registryOut field on the push configuration object.
func WithPushOptWriter(out io.Writer) PushOpt {
	return func(p *Push) {
		p.out = out
	}
}

// WithPushRegistryClient sets the registry client on the push configuration object.
func WithPushRegistryClient(client *registry.Client) PushOpt {
	return func(p *Push) {
		p.cfg.RegistryClient = client
	}
}

// NewPush creates a new push, with configuration options.
func NewPush(settings *config.GlobalSettings, cfg *registry.Configuration, opts ...any) *Push {
	p := &Push{
		Settings: settings,
		cfg:      cfg,
	}
	for _, opt := range opts {
		switch r := opt.(type) {
		case PushOpt:
			r(p)
		case registry.RegistryParamsOpt:
			r(&p.RegistryParams)
		default:
			panic("unknown type passed to NewPush")
		}
	}
	return p
}

// Run executes the publish action.
func (p *Push) Run(wasmExe string, metadataFile string, remote string) (string, error) {
	var out strings.Builder

	if !registry.IsOCI(remote) {
		return "", fmt.Errorf("only OCI registries are supported")
	}

	c := WASMUploader{
		Out:     &out,
		Pushers: All(p.Settings),
		Options: []Option{
			WithTLSClientConfig(p.CertFile, p.KeyFile, p.CAFile),
			WithInsecureSkipTLSVerify(p.Insecure),
			WithPlainHTTP(p.PlainHTTP),
			WithRegistryClient(p.cfg.RegistryClient),
		},
	}

	return out.String(), c.UploadTo(wasmExe, metadataFile, remote)
}
