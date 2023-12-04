package downloader

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	"github.com/inercia/proxy-wasm-oci/pkg/registry"
	"github.com/inercia/proxy-wasm-oci/pkg/utils"
)

// VerificationStrategy describes a strategy for determining whether to verify a chart.
type VerificationStrategy int

const (
	// VerifyNever will skip all verification of a chart.
	VerifyNever VerificationStrategy = iota
	// VerifyIfPossible will attempt a verification, it will not error if verification
	// data is missing. But it will not stop processing if verification fails.
	VerifyIfPossible
	// VerifyAlways will always attempt a verification, and will fail if the
	// verification fails.
	VerifyAlways
	// VerifyLater will fetch verification data, but not do any verification.
	// This is to accommodate the case where another step of the process will
	// perform verification.
	VerifyLater
)

type Verification string

// ErrNoOwnerRepo indicates that a given chart URL can't be found in any repos.
var ErrNoOwnerRepo = errors.New("could not find a repo containing the given URL")

// WASMDownloader handles downloading a chart.
//
// It is capable of performing verifications on charts as well.
type WASMDownloader struct {
	// Out is the location to write warning and info messages.
	Out io.Writer
	// Verify indicates what verification strategy to use.
	Verify VerificationStrategy
	// Getter collection for the operation
	Getters Providers
	// Options provide parameters to be passed along to the Getter being initialized.
	Options        []Option
	RegistryClient *registry.Client
}

// DownloadTo retrieves a WASM extension.
//
// Returns a string path to the location where the file was downloaded and a verification
// (if provenance was verified), or an error if something bad happened.
func (c *WASMDownloader) DownloadTo(ref, version, dest string) (string, *Verification, error) {
	u, err := c.ResolveWASMExtVersion(ref, version)
	if err != nil {
		return "", nil, err
	}

	g, err := c.Getters.ByScheme(u.Scheme)
	if err != nil {
		return "", nil, err
	}

	data, err := g.Get(u.String(), c.Options...)
	if err != nil {
		return "", nil, err
	}

	name := filepath.Base(u.Path)
	idx := strings.LastIndexByte(name, ':')
	name = fmt.Sprintf("%s-%s.wasm", name[:idx], name[idx+1:])

	destfile := filepath.Join(dest, name)
	if err := utils.AtomicWriteFile(destfile, data, 0o644); err != nil {
		return destfile, nil, err
	}

	verification := Verification("")
	if c.Verify > VerifyNever {
		// TODO
	}

	return destfile, &verification, nil
}

func (c *WASMDownloader) getOciURI(ref, version string, u *url.URL) (*url.URL, error) {
	var tag string
	var err error

	// Evaluate whether an explicit version has been provided. Otherwise, determine version to use
	_, errSemVer := semver.NewVersion(version)
	if errSemVer == nil {
		tag = version
	} else {
		// Retrieve list of repository tags
		tags, err := c.RegistryClient.Tags(strings.TrimPrefix(ref, fmt.Sprintf("%s://", registry.OCIScheme)))
		if err != nil {
			return nil, err
		}
		if len(tags) == 0 {
			return nil, errors.Errorf("Unable to locate any tags in provided repository: %s", ref)
		}

		// Determine if version provided
		// If empty, try to get the highest available tag
		// If exact version, try to find it
		// If semver constraint string, try to find a match
		tag, err = registry.GetTagMatchingVersionOrConstraint(tags, version)
		if err != nil {
			return nil, err
		}
	}

	comps := strings.Split(u.Path, ":")
	u.Path = fmt.Sprintf("%s:%s", comps[0], tag)

	return u, err
}

// ResolveWASMExtVersion resolves a chart reference to a URL.
//
// It returns the URL and sets the ChartDownloader's Options that can fetch
// the URL using the appropriate Getter.
//
// A reference may be an HTTP URL, an oci reference URL, a 'reponame/chartname'
// reference, or a local path.
//
// A version is a SemVer string (1.2.3-beta.1+f334a6789).
//
//   - For fully qualified URLs, the version will be ignored (since URLs aren't versioned)
//   - For a chart reference
//   - If version is non-empty, this will return the URL for that version
//   - If version is empty, this will return the URL for the latest version
//   - If no version can be found, an error is returned
func (c *WASMDownloader) ResolveWASMExtVersion(ref, version string) (*url.URL, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return nil, errors.Errorf("invalid chart URL format: %s", ref)
	}

	if !registry.IsOCI(u.String()) {
		return nil, errors.Errorf("invalid URL: not an OCI", ref)
	}

	return c.getOciURI(ref, version, u)
}

// isTar tests whether the given file is a tar file.
//
// Currently, this simply checks extension, since a subsequent function will
// untar the file and validate its binary format.
func isTar(filename string) bool {
	return strings.EqualFold(filepath.Ext(filename), ".tgz")
}
