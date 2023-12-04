package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"oras.land/oras-go/pkg/auth"
	dockerauth "oras.land/oras-go/pkg/auth/docker"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
	"oras.land/oras-go/pkg/registry"
	registryremote "oras.land/oras-go/pkg/registry/remote"
	registryauth "oras.land/oras-go/pkg/registry/remote/auth"

	"github.com/inercia/proxy-wasm-oci/pkg/common"
	"github.com/inercia/proxy-wasm-oci/pkg/config"
	"github.com/inercia/proxy-wasm-oci/pkg/version"
)

type (
	// Client works with OCI-compliant registries
	Client struct {
		debug       bool
		enableCache bool
		// path to repository config file e.g. ~/.docker/config.json
		credentialsFile    string
		out                io.Writer
		authorizer         auth.Client
		registryAuthorizer *registryauth.Client
		resolver           func(ref registry.Reference) (remotes.Resolver, error)
		httpClient         *http.Client
		plainHTTP          bool
	}

	// ClientOption allows specifying various settings configurable by the user for overriding the defaults
	// used when creating a new default client
	ClientOption func(*Client)
)

// NewClient returns a new registry client with config
func NewClient(options ...ClientOption) (*Client, error) {
	client := &Client{
		out: io.Discard,
	}
	for _, option := range options {
		option(client)
	}
	if client.credentialsFile == "" {
		client.credentialsFile = config.ConfigPath(CredentialsFileBasename)
	}
	if client.authorizer == nil {
		authClient, err := dockerauth.NewClientWithDockerFallback(client.credentialsFile)
		if err != nil {
			return nil, err
		}
		client.authorizer = authClient
	}

	resolverFn := client.resolver // copy for avoiding recursive call
	client.resolver = func(ref registry.Reference) (remotes.Resolver, error) {
		if resolverFn != nil {
			// validate if the resolverFn returns a valid resolver
			if resolver, err := resolverFn(ref); resolver != nil && err == nil {
				return resolver, nil
			}
		}
		headers := http.Header{}
		headers.Set("User-Agent", version.GetUserAgent())
		opts := []auth.ResolverOption{auth.WithResolverHeaders(headers)}
		if client.httpClient != nil {
			opts = append(opts, auth.WithResolverClient(client.httpClient))
		}
		if client.plainHTTP {
			opts = append(opts, auth.WithResolverPlainHTTP())
		}
		resolver, err := client.authorizer.ResolverWithOpts(opts...)
		if err != nil {
			return nil, err
		}
		return resolver, nil
	}

	// allocate a cache if option is set
	var cache registryauth.Cache
	if client.enableCache {
		cache = registryauth.DefaultCache
	}
	if client.registryAuthorizer == nil {
		client.registryAuthorizer = &registryauth.Client{
			Client: client.httpClient,
			Header: http.Header{
				"User-Agent": {version.GetUserAgent()},
			},
			Cache: cache,
			Credential: func(ctx context.Context, reg string) (registryauth.Credential, error) {
				dockerClient, ok := client.authorizer.(*dockerauth.Client)
				if !ok {
					return registryauth.EmptyCredential, errors.New("unable to obtain docker client")
				}

				username, password, err := dockerClient.Credential(reg)
				if err != nil {
					return registryauth.EmptyCredential, errors.New("unable to retrieve credentials")
				}

				// A blank returned username and password value is a bearer token
				if username == "" && password != "" {
					return registryauth.Credential{
						RefreshToken: password,
					}, nil
				}

				return registryauth.Credential{
					Username: username,
					Password: password,
				}, nil
			},
		}
	}
	return client, nil
}

func NewDefaultRegistryClient(plainHTTP bool) (*Client, error) {
	opts := []ClientOption{
		ClientOptEnableCache(true),
		ClientOptWriter(os.Stderr),
	}
	if plainHTTP {
		opts = append(opts, ClientOptPlainHTTP())
	}

	// Create a new registry client
	registryClient, err := NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

func NewClientWithParams(p RegistryParams, registryConfig string, debug bool) (*Client, error) {
	if p.PlainHTTP {
		registryClient, err := NewDefaultRegistryClient(p.PlainHTTP)
		if err != nil {
			return nil, err
		}
		return registryClient, nil
	}

	if p.CertFile != "" && p.KeyFile != "" || p.CAFile != "" || p.Insecure {
		registryClient, err := NewClientWithTLSWithParams(p, registryConfig, debug)
		if err != nil {
			return nil, err
		}

		return registryClient, nil
	}

	return NewDefaultRegistryClient(p.PlainHTTP)
}

func NewClientWithTLSWithParams(p RegistryParams, registryConfig string, debug bool) (*Client, error) {
	// Create a new registry client
	registryClient, err := NewRegistryClientWithTLS(os.Stderr, p, registryConfig, debug)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

// ClientOptDebug returns a function that sets the debug setting on client options set
func ClientOptDebug(debug bool) ClientOption {
	return func(client *Client) {
		client.debug = debug
	}
}

// ClientOptEnableCache returns a function that sets the enableCache setting on a client options set
func ClientOptEnableCache(enableCache bool) ClientOption {
	return func(client *Client) {
		client.enableCache = enableCache
	}
}

// ClientOptWriter returns a function that sets the writer setting on client options set
func ClientOptWriter(out io.Writer) ClientOption {
	return func(client *Client) {
		client.out = out
	}
}

// ClientOptCredentialsFile returns a function that sets the credentialsFile setting on a client options set
func ClientOptCredentialsFile(credentialsFile string) ClientOption {
	return func(client *Client) {
		client.credentialsFile = credentialsFile
	}
}

// ClientOptHTTPClient returns a function that sets the httpClient setting on a client options set
func ClientOptHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

func ClientOptPlainHTTP() ClientOption {
	return func(c *Client) {
		c.plainHTTP = true
	}
}

// ClientOptResolver returns a function that sets the resolver setting on a client options set
func ClientOptResolver(resolver remotes.Resolver) ClientOption {
	return func(client *Client) {
		client.resolver = func(ref registry.Reference) (remotes.Resolver, error) {
			return resolver, nil
		}
	}
}

///////////////////////////////////////////////////////////////////////
// login operations
///////////////////////////////////////////////////////////////////////

type (
	// LoginOption allows specifying various settings on login
	LoginOption func(*loginOperation)

	loginOperation struct {
		username string
		password string
		insecure bool
		certFile string
		keyFile  string
		caFile   string
	}
)

// Login logs into a registry
func (c *Client) Login(host string, options ...LoginOption) error {
	operation := &loginOperation{}
	for _, option := range options {
		option(operation)
	}
	authorizerLoginOpts := []auth.LoginOption{
		auth.WithLoginContext(ctx(c.out, c.debug)),
		auth.WithLoginHostname(host),
		auth.WithLoginUsername(operation.username),
		auth.WithLoginSecret(operation.password),
		auth.WithLoginUserAgent(version.GetUserAgent()),
		auth.WithLoginTLS(operation.certFile, operation.keyFile, operation.caFile),
	}
	if operation.insecure {
		authorizerLoginOpts = append(authorizerLoginOpts, auth.WithLoginInsecure())
	}
	if err := c.authorizer.LoginWithOpts(authorizerLoginOpts...); err != nil {
		return err
	}
	fmt.Fprintln(c.out, "Login Succeeded")
	return nil
}

// LoginOptBasicAuth returns a function that sets the username/password settings on login
func LoginOptBasicAuth(username string, password string) LoginOption {
	return func(operation *loginOperation) {
		operation.username = username
		operation.password = password
	}
}

// LoginOptInsecure returns a function that sets the insecure setting on login
func LoginOptInsecure(insecure bool) LoginOption {
	return func(operation *loginOperation) {
		operation.insecure = insecure
	}
}

// LoginOptTLSClientConfig returns a function that sets the TLS settings on login.
func LoginOptTLSClientConfig(certFile, keyFile, caFile string) LoginOption {
	return func(operation *loginOperation) {
		operation.certFile = certFile
		operation.keyFile = keyFile
		operation.caFile = caFile
	}
}

type (
	// LogoutOption allows specifying various settings on logout
	LogoutOption func(*logoutOperation)

	logoutOperation struct{}
)

// Logout logs out of a registry
func (c *Client) Logout(host string, opts ...LogoutOption) error {
	operation := &logoutOperation{}
	for _, opt := range opts {
		opt(operation)
	}
	if err := c.authorizer.Logout(ctx(c.out, c.debug), host); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "Removing login credentials for %s\n", host)
	return nil
}

///////////////////////////////////////////////////////////////////////
// pull operations
///////////////////////////////////////////////////////////////////////

type (
	// PullOption allows specifying various settings on pull
	PullOption func(*pullOperation)

	// PullResult is the result returned upon successful pull.
	PullResult struct {
		Manifest *DescriptorPullSummary         `json:"manifest"`
		Config   *DescriptorPullSummary         `json:"config"`
		WASMExt  *DescriptorPullSummaryWithMeta `json:"wasm"`
		Ref      string                         `json:"ref"`
	}

	DescriptorPullSummary struct {
		Data   []byte `json:"-"`
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	}

	DescriptorPullSummaryWithMeta struct {
		DescriptorPullSummary
		Meta *common.Metadata `json:"meta"`
	}

	pullOperation struct{}
)

// Pull downloads a WASM extension from a registry
func (c *Client) Pull(ref string, options ...PullOption) (*PullResult, error) {
	parsedRef, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	operation := &pullOperation{}
	for _, option := range options {
		option(operation)
	}
	memoryStore := content.NewMemory()
	allowedMediaTypes := []string{
		WASMMetadataMediaType,
	}
	minNumDescriptors := 1 // 1 for the config
	minNumDescriptors++
	allowedMediaTypes = append(allowedMediaTypes, WASMLayerMediaType)

	var descriptors, layers []ocispec.Descriptor
	remotesResolver, err := c.resolver(parsedRef)
	if err != nil {
		return nil, err
	}
	registryStore := content.Registry{Resolver: remotesResolver}

	manifest, err := oras.Copy(ctx(c.out, c.debug), registryStore, parsedRef.String(), memoryStore, "",
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(allowedMediaTypes),
		oras.WithLayerDescriptors(func(l []ocispec.Descriptor) {
			layers = l
		}))
	if err != nil {
		return nil, err
	}

	descriptors = append(descriptors, manifest)
	descriptors = append(descriptors, layers...)

	numDescriptors := len(descriptors)
	if numDescriptors < minNumDescriptors {
		return nil, fmt.Errorf("manifest does not contain minimum number of descriptors (%d), descriptors found: %d",
			minNumDescriptors, numDescriptors)
	}
	var configDescriptor *ocispec.Descriptor
	var wasmDescriptor *ocispec.Descriptor
	for _, descriptor := range descriptors {
		d := descriptor
		switch d.MediaType {
		case WASMMetadataMediaType:
			configDescriptor = &d
		case WASMLayerMediaType:
			wasmDescriptor = &d
		}
	}
	if configDescriptor == nil {
		return nil, fmt.Errorf("could not load config with mediatype %s", WASMMetadataMediaType)
	}
	if wasmDescriptor == nil {
		return nil, fmt.Errorf("manifest does not contain a layer with mediatype %s",
			WASMLayerMediaType)
	}

	result := &PullResult{
		Manifest: &DescriptorPullSummary{
			Digest: manifest.Digest.String(),
			Size:   manifest.Size,
		},
		Config: &DescriptorPullSummary{
			Digest: configDescriptor.Digest.String(),
			Size:   configDescriptor.Size,
		},
		WASMExt: &DescriptorPullSummaryWithMeta{},
		Ref:     parsedRef.String(),
	}

	var getManifestErr error
	if _, manifestData, ok := memoryStore.Get(manifest); !ok {
		getManifestErr = errors.Errorf("Unable to retrieve blob with digest %s", manifest.Digest)
	} else {
		result.Manifest.Data = manifestData
	}
	if getManifestErr != nil {
		return nil, getManifestErr
	}
	var getConfigDescriptorErr error
	if _, configData, ok := memoryStore.Get(*configDescriptor); !ok {
		getConfigDescriptorErr = errors.Errorf("Unable to retrieve blob with digest %s", configDescriptor.Digest)
	} else {
		result.Config.Data = configData
		var meta *common.Metadata
		if err := json.Unmarshal(configData, &meta); err != nil {
			return nil, err
		}
		result.WASMExt.Meta = meta
	}
	if getConfigDescriptorErr != nil {
		return nil, getConfigDescriptorErr
	}

	var getWASMExtDescriptorErr error
	if _, wasmData, ok := memoryStore.Get(*wasmDescriptor); !ok {
		getWASMExtDescriptorErr = errors.Errorf("Unable to retrieve blob with digest %s", wasmDescriptor.Digest)
	} else {
		result.WASMExt.Data = wasmData
		result.WASMExt.Digest = wasmDescriptor.Digest.String()
		result.WASMExt.Size = wasmDescriptor.Size
	}
	if getWASMExtDescriptorErr != nil {
		return nil, getWASMExtDescriptorErr
	}

	fmt.Fprintf(c.out, "Pulled: %s\n", result.Ref)
	fmt.Fprintf(c.out, "Digest: %s\n", result.Manifest.Digest)

	if strings.Contains(result.Ref, "_") {
		fmt.Fprintf(c.out, "%s contains an underscore.\n", result.Ref)
	}

	return result, nil
}

///////////////////////////////////////////////////////////////////////
// push operations
///////////////////////////////////////////////////////////////////////

type (
	// PushOption allows specifying various settings on push
	PushOption func(*pushOperation)

	// PushResult is the result returned upon successful push.
	PushResult struct {
		Manifest *descriptorPushSummary         `json:"manifest"`
		Config   *descriptorPushSummary         `json:"config"`
		WASMExt  *descriptorPushSummaryWithMeta `json:"wasm"`
		Ref      string                         `json:"ref"`
	}

	descriptorPushSummary struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	}

	descriptorPushSummaryWithMeta struct {
		descriptorPushSummary
		Meta *common.Metadata `json:"meta"`
	}

	pushOperation struct {
		strictMode bool
		test       bool
	}
)

// Push uploads a chart to a registry.
func (c *Client) Push(data []byte, meta common.Metadata, ref string, options ...PushOption) (*PushResult, error) {
	parsedRef, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	operation := &pushOperation{
		strictMode: true, // By default, enable strict mode
	}
	for _, option := range options {
		option(operation)
	}

	memoryStore := content.NewMemory()
	wasmExeDescriptor, err := memoryStore.Add("", WASMLayerMediaType, data)
	if err != nil {
		return nil, err
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	metaDescriptor, err := memoryStore.Add("", WASMMetadataMediaType, metaBytes)
	if err != nil {
		return nil, err
	}

	descriptors := []ocispec.Descriptor{wasmExeDescriptor}

	ociAnnotations := generateOCIAnnotations(&meta, operation.test)

	manifestData, manifest, err := content.GenerateManifest(&metaDescriptor, ociAnnotations, descriptors...)
	if err != nil {
		return nil, err
	}

	if err := memoryStore.StoreManifest(parsedRef.String(), manifest, manifestData); err != nil {
		return nil, err
	}

	remotesResolver, err := c.resolver(parsedRef)
	if err != nil {
		return nil, err
	}
	registryStore := content.Registry{Resolver: remotesResolver}
	_, err = oras.Copy(ctx(c.out, c.debug), memoryStore, parsedRef.String(), registryStore, "",
		oras.WithNameValidation(nil))
	if err != nil {
		return nil, err
	}

	wasmSummary := &descriptorPushSummaryWithMeta{
		Meta: &meta,
	}
	wasmSummary.Digest = wasmExeDescriptor.Digest.String()
	wasmSummary.Size = wasmExeDescriptor.Size
	result := &PushResult{
		Manifest: &descriptorPushSummary{
			Digest: manifest.Digest.String(),
			Size:   manifest.Size,
		},
		Config: &descriptorPushSummary{
			Digest: metaDescriptor.Digest.String(),
			Size:   metaDescriptor.Size,
		},
		WASMExt: wasmSummary,
		Ref:     parsedRef.String(),
	}
	fmt.Fprintf(c.out, "Pushed: %s\n", result.Ref)
	fmt.Fprintf(c.out, "Digest: %s\n", result.Manifest.Digest)
	if strings.Contains(parsedRef.Reference, "_") {
		fmt.Fprintf(c.out, "%s contains an underscore.\n", result.Ref)
	}

	return result, err
}

// PushOptStrictMode returns a function that sets the strictMode setting on push
func PushOptStrictMode(strictMode bool) PushOption {
	return func(operation *pushOperation) {
		operation.strictMode = strictMode
	}
}

// PushOptTest returns a function that sets whether test setting on push
func PushOptTest(test bool) PushOption {
	return func(operation *pushOperation) {
		operation.test = test
	}
}

///////////////////////////////////////////////////////////////////////
// other operations
///////////////////////////////////////////////////////////////////////

// Tags provides a sorted list all semver compliant tags for a given repository
func (c *Client) Tags(ref string) ([]string, error) {
	parsedReference, err := registry.ParseReference(ref)
	if err != nil {
		return nil, err
	}

	repository := registryremote.Repository{
		Reference: parsedReference,
		Client:    c.registryAuthorizer,
		PlainHTTP: c.plainHTTP,
	}

	var registryTags []string

	registryTags, err = registry.Tags(ctx(c.out, c.debug), &repository)
	if err != nil {
		return nil, err
	}

	var tagVersions []*semver.Version
	for _, tag := range registryTags {
		// Change underscore (_) back to plus (+) for Helm
		// See https://github.com/helm/helm/issues/10166
		tagVersion, err := semver.StrictNewVersion(strings.ReplaceAll(tag, "_", "+"))
		if err == nil {
			tagVersions = append(tagVersions, tagVersion)
		}
	}

	// Sort the collection
	sort.Sort(sort.Reverse(semver.Collection(tagVersions)))

	tags := make([]string, len(tagVersions))

	for iTv, tv := range tagVersions {
		tags[iTv] = tv.String()
	}

	return tags, nil
}
