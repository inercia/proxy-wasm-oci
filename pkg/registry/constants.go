package registry

const (
	// OCIScheme is the URL scheme for OCI-based requests
	OCIScheme = "oci"

	// CredentialsFileBasename is the filename for auth credentials file
	CredentialsFileBasename = "registry/config.json"

	// WASMMetadataMediaType is the reserved media type for the metadata
	WASMMetadataMediaType = "application/vnd.wasm.config.v1+json"

	// WASMLayerMediaType is the reserved media type for Proxy Wasm Publisher package content
	WASMLayerMediaType = "application/vnd.wasm.content.layer.v1+wasm"
)
