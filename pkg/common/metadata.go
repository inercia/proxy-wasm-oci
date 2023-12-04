package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Masterminds/semver/v3"
	"sigs.k8s.io/yaml"
)

// Maintainer describes a WASM extension maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty"`
	// URL is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty"`
}

// Validate checks valid data and sanitizes string characters.
func (m *Maintainer) Validate() error {
	if m == nil {
		return ValidationError("maintainers must not contain empty or null nodes")
	}
	m.Name = sanitizeString(m.Name)
	m.Email = sanitizeString(m.Email)
	m.URL = sanitizeString(m.URL)
	return nil
}

// Metadata for a proxy-WASM extension. This models the structure of a Wasm.yaml file.
type Metadata struct {
	// The name of the WASM extension. Required.
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this WASM extension
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the WASM extension. Required.
	Version string `json:"version,omitempty"`
	// A one-sentence description of the WASM extension
	Description string `json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
	// The API Version of this WASM extension. Required.
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable WASM extension
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable WASM extension
	Tags string `json:"tags,omitempty"`
	// Whether or not this WASM extension is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings made available for inspection by other applications.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Specifies the WASM extension type.
	Type string `json:"type,omitempty"`
}

func NewMetadataFromFile(filename string) (Metadata, error) {
	extension := filepath.Ext(filename)

	var metadata Metadata

	switch extension {
	case ".yaml", ".yml":
		data, err := os.ReadFile(filename)
		if err != nil {
			return metadata, err
		}

		err = yaml.Unmarshal(data, &metadata)
		if err != nil {
			return metadata, err
		}
	case ".json":
		data, err := os.ReadFile(filename)
		if err != nil {
			return metadata, err
		}

		err = json.Unmarshal(data, &metadata)
		if err != nil {
			return metadata, err
		}

	default:
		return metadata, ValidationError("unsupported file extension")
	}

	return metadata, nil
}

// Validate checks the metadata for known issues and sanitizes string
// characters.
func (md *Metadata) Validate() error {
	if md == nil {
		return ValidationError("metadata is required")
	}

	md.Name = sanitizeString(md.Name)
	md.Description = sanitizeString(md.Description)
	md.Home = sanitizeString(md.Home)
	md.Icon = sanitizeString(md.Icon)
	md.Condition = sanitizeString(md.Condition)
	md.Tags = sanitizeString(md.Tags)
	for i := range md.Sources {
		md.Sources[i] = sanitizeString(md.Sources[i])
	}
	for i := range md.Keywords {
		md.Keywords[i] = sanitizeString(md.Keywords[i])
	}

	if md.APIVersion == "" {
		return ValidationError("'apiVersion' is required")
	}
	if md.Name == "" {
		return ValidationError("'name' is required")
	}
	if md.Version == "" {
		return ValidationError("'version' is required")
	}
	if !isValidSemver(md.Version) {
		return ValidationErrorf("'version' %q is invalid", md.Version)
	}

	for _, m := range md.Maintainers {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func isValidWASMType(in string) bool {
	switch in {
	case "", "authn", "authz":
		return true
	}
	return false
}

func isValidSemver(v string) bool {
	_, err := semver.NewVersion(v)
	return err == nil
}

// sanitizeString normalize spaces and removes non-printable characters.
func sanitizeString(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, str)
}
