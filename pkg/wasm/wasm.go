package wasm

import (
	"regexp"

	"github.com/inercia/proxy-wasm-oci/pkg/common"
)

// APIVersionV1 is the API version number for version 1.
const APIVersionV1 = "v1"

// APIVersionV2 is the API version number for version 2.
const APIVersionV2 = "v2"

// aliasNameFormat defines the characters that are legal in an alias name.
var aliasNameFormat = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// File represents a file as a name/value pair.
//
// By convention, name is a relative path within the scope of the chart's
// base directory.
type File struct {
	// Name is the path-like name of the template.
	Name string `json:"name"`
	// Data is the template as byte data.
	Data []byte `json:"data"`
}

// WASMExtension is a Proxy-WASM extension that contains metadata, a default config, zero or more
// optionally parameterizable templates.
type WASMExtension struct {
	// Raw contains the raw contents of the files originally contained in the chart archive.
	//
	// This should not be used except in special cases like `helm show values`,
	// where we want to display the raw values, comments and all.
	Raw []*File `json:"-"`
	// Metadata is the contents of the Wasmfile.
	Metadata *common.Metadata `json:"metadata"`
	// Templates for this chart.
	Templates []*File `json:"templates"`
	// Files are miscellaneous files in a chart archive,
	// e.g. README, LICENSE, etc.
	Files []*File `json:"files"`
}

// Name returns the name of the Proxy-WASM Extension.
func (ch *WASMExtension) Name() string {
	if ch.Metadata == nil {
		return ""
	}
	return ch.Metadata.Name
}

// Validate validates the metadata.
func (ch *WASMExtension) Validate() error {
	return ch.Metadata.Validate()
}
