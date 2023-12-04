package wasm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"

	"sigs.k8s.io/yaml"

	"github.com/inercia/proxy-wasm-oci/pkg/common"
)

var drivePathPattern = regexp.MustCompile(`^[a-zA-Z]:/`)

// FileLoader loads a chart from a file
type FileLoader string

// Load loads a chart
func (l FileLoader) Load() (*WASMExtension, error) {
	return LoadFile(string(l))
}

// LoadFile loads from an archive file.
func LoadFile(name string) (*WASMExtension, error) {
	if fi, err := os.Stat(name); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("cannot load a directory")
	}

	metadataFile := path.Join(path.Dir(name), "Wasm.yaml")
	if fi, err := os.Stat(metadataFile); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("cannot load a metadata file")
	}

	raw, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer raw.Close()

	c, err := LoadWASM(raw, metadataFile)
	if err != nil {
		return nil, err
	}
	return c, err
}

// LoadWASMFromReader reads in files out of an archive into memory. This function
// performs important path security checks and should always be used before
// expanding a tarball
func LoadWASMFromReader(in io.Reader, metadataFile string) ([]*BufferedFile, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	metadata, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, err
	}

	files := []*BufferedFile{
		{Name: "main.wasm", Data: data},
		{Name: "Wasm.yaml", Data: metadata},
	}
	return files, nil
}

// LoadWASM loads from a reader containing.
func LoadWASM(in io.Reader, metadata string) (*WASMExtension, error) {
	files, err := LoadWASMFromReader(in, metadata)
	if err != nil {
		return nil, err
	}

	return LoadFiles(files)
}

// WASMExtensionLoader loads a chart.
type WASMExtensionLoader interface {
	Load() (*WASMExtension, error)
}

// Loader returns a new ChartLoader appropriate for the given WASM extension name
func Loader(name string) (WASMExtensionLoader, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("could not load directory %q", name)
	}
	return FileLoader(name), nil

}

// Load takes a string name, tries to resolve it to a file or directory, and then loads it.
//
// This is the preferred way to load a chart. It will discover the chart encoding
// and hand off to the appropriate chart reader.
//
// If a .helmignore file is present, the directory loader will skip loading any files
// matching it. But .helmignore is not evaluated when reading out of an archive.
func Load(name string) (*WASMExtension, error) {
	l, err := Loader(name)
	if err != nil {
		return nil, err
	}
	return l.Load()
}

// BufferedFile represents an archive file buffered for later processing.
type BufferedFile struct {
	Name string
	Data []byte
}

// LoadFiles loads from in-memory files.
func LoadFiles(files []*BufferedFile) (*WASMExtension, error) {
	c := new(WASMExtension)

	// do not rely on assumed ordering of files in the chart and crash
	// if Wasm.yaml was not coming early enough to initialize metadata
	for _, f := range files {
		c.Raw = append(c.Raw, &File{Name: f.Name, Data: f.Data})
		if f.Name == "Wasm.yaml" {
			if c.Metadata == nil {
				c.Metadata = new(common.Metadata)
			}
			if err := yaml.Unmarshal(f.Data, c.Metadata); err != nil {
				return c, fmt.Errorf("cannot load Wasm.yaml: %w", err)
			}
			// NOTE(bacongobbler): while the chart specification says that APIVersion must be set,
			// Helm 2 accepted charts that did not provide an APIVersion in their chart metadata.
			// Because of that, if APIVersion is unset, we should assume we're loading a v1 chart.
			if c.Metadata.APIVersion == "" {
				c.Metadata.APIVersion = APIVersionV1
			}
		}
	}

	for _, f := range files {
		switch {
		case f.Name == "Wasm.yaml":
			// already processed
			continue
		default:
			c.Files = append(c.Files, &File{Name: f.Name, Data: f.Data})
		}
	}

	if c.Metadata == nil {
		return c, errors.New("Wasm.yaml file is missing")
	}

	if err := c.Validate(); err != nil {
		return c, err
	}

	return c, nil
}
