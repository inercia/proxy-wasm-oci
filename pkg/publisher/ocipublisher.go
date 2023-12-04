package publisher

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/inercia/proxy-wasm-oci/pkg/common"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

// OCIPusher is the default OCI backend handler
type OCIPusher struct {
	opts options
}

// Push performs a Push from repo.Pusher.
func (pusher *OCIPusher) Push(wasmExe, metadataFile, href string, options ...Option) error {
	for _, opt := range options {
		opt(&pusher.opts)
	}
	return pusher.push(wasmExe, metadataFile, href)
}

func (pusher *OCIPusher) push(wasmExe, metadataFile, href string) error {
	stat, err := os.Stat(wasmExe)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: no such file", wasmExe)
		}
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("cannot push directory, must provide Proxy-WASM file")
	}

	meta, err := common.NewMetadataFromFile(metadataFile)
	if err != nil {
		return err
	}

	client := pusher.opts.registryClient
	if client == nil {
		c, err := pusher.newRegistryClient()
		if err != nil {
			return err
		}
		client = c
	}

	wasmExeBytes, err := os.ReadFile(wasmExe)
	if err != nil {
		return err
	}

	var pushOpts []registry.PushOption

	ref := fmt.Sprintf("%s:%s",
		path.Join(strings.TrimPrefix(href, fmt.Sprintf("%s://", registry.OCIScheme)), meta.Name),
		meta.Version)

	_, err = client.Push(wasmExeBytes, meta, ref, pushOpts...)
	return err
}

// NewOCIPusher constructs a valid OCI client as a Pusher
func NewOCIPusher(ops ...Option) (Pusher, error) {
	var client OCIPusher

	for _, opt := range ops {
		opt(&client.opts)
	}

	return &client, nil
}

func (pusher *OCIPusher) newRegistryClient() (*registry.Client, error) {
	if (pusher.opts.CertFile != "" && pusher.opts.KeyFile != "") || pusher.opts.CAFile != "" || pusher.opts.Insecure {
		tlsConf, err := registry.NewClientTLS(pusher.opts.RegistryParams)
		if err != nil {
			return nil, fmt.Errorf("can't create TLS config for client: %w", err)
		}

		registryClient, err := registry.NewClient(
			registry.ClientOptHTTPClient(&http.Client{
				// From https://github.com/google/go-containerregistry/blob/31786c6cbb82d6ec4fb8eb79cd9387905130534e/pkg/v1/remote/options.go#L87
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					DialContext: (&net.Dialer{
						// By default we wrap the transport in retries, so reduce the
						// default dial timeout to 5s to avoid 5x 30s of connection
						// timeouts when doing the "ping" on certain http registries.
						Timeout:   5 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					ForceAttemptHTTP2:     true,
					MaxIdleConns:          100,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					TLSClientConfig:       tlsConf,
				},
			}),
			registry.ClientOptEnableCache(true),
		)
		if err != nil {
			return nil, err
		}
		return registryClient, nil
	}

	opts := []registry.ClientOption{registry.ClientOptEnableCache(true)}
	if pusher.opts.PlainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}
