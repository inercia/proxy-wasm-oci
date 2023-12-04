package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	orascontext "oras.land/oras-go/pkg/context"
	"oras.land/oras-go/pkg/registry"

	"github.com/inercia/proxy-wasm-oci/pkg/common"
	"github.com/inercia/proxy-wasm-oci/pkg/utils"
)

var immutableOciAnnotations = []string{
	ocispec.AnnotationVersion,
	ocispec.AnnotationTitle,
}

// IsOCI determines whether or not a URL is to be treated as an OCI URL
func IsOCI(url string) bool {
	return strings.HasPrefix(url, fmt.Sprintf("%s://", OCIScheme))
}

// ContainsTag determines whether a tag is found in a provided list of tags
func ContainsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if tag == t {
			return true
		}
	}
	return false
}

func GetTagMatchingVersionOrConstraint(tags []string, versionString string) (string, error) {
	var constraint *semver.Constraints
	if versionString == "" {
		// If string is empty, set wildcard constraint
		constraint, _ = semver.NewConstraint("*")
	} else {
		// when customer input exact version, check whether have exact match
		// one first
		for _, v := range tags {
			if versionString == v {
				return v, nil
			}
		}

		// Otherwise set constraint to the string given
		var err error
		constraint, err = semver.NewConstraint(versionString)
		if err != nil {
			return "", err
		}
	}

	// Otherwise try to find the first available version matching the string,
	// in case it is a constraint
	for _, v := range tags {
		test, err := semver.NewVersion(v)
		if err != nil {
			continue
		}
		if constraint.Check(test) {
			return v, nil
		}
	}

	return "", errors.Errorf("Could not locate a version matching provided version string %s", versionString)
}

// ctx retrieves a fresh context.
// disable verbose logging coming from ORAS (unless debug is enabled)
func ctx(out io.Writer, debug bool) context.Context {
	if !debug {
		return orascontext.Background()
	}
	ctx := orascontext.WithLoggerFromWriter(context.Background(), out)
	orascontext.GetLogger(ctx).Logger.SetLevel(logrus.DebugLevel)
	return ctx
}

// parseReference will parse and validate the reference, and clean tags when
// applicable tags are only cleaned when plus (+) signs are present, and are
// converted to underscores (_) before pushing
// See https://github.com/helm/helm/issues/10166
func parseReference(raw string) (registry.Reference, error) {
	// The sole possible reference modification is replacing plus (+) signs
	// present in tags with underscores (_). To do this properly, we first
	// need to identify a tag, and then pass it on to the reference parser
	// NOTE: Passing immediately to the reference parser will fail since (+)
	// signs are an invalid tag character, and simply replacing all plus (+)
	// occurrences could invalidate other portions of the URI
	parts := strings.Split(raw, ":")
	if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "/") {
		tag := parts[len(parts)-1]

		if tag != "" {
			// Replace any plus (+) signs with known underscore (_) conversion
			newTag := strings.ReplaceAll(tag, "+", "_")
			raw = strings.ReplaceAll(raw, tag, newTag)
		}
	}

	return registry.ParseReference(raw)
}

// NewRegistryClientWithTLS is a helper function to create a new registry client with TLS enabled.
func NewRegistryClientWithTLS(out io.Writer, p RegistryParams, registryConfig string, debug bool) (*Client, error) {
	tlsConf, err := NewClientTLS(p)
	if err != nil {
		return nil, fmt.Errorf("can't create TLS config for client: %s", err)
	}
	// Create a new registry client
	registryClient, err := NewClient(
		ClientOptDebug(debug),
		ClientOptEnableCache(true),
		ClientOptWriter(out),
		ClientOptCredentialsFile(registryConfig),
		ClientOptHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			},
		}),
	)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}

// generateOCIAnnotations will generate OCI annotations to include within the OCI manifest
func generateOCIAnnotations(meta *common.Metadata, test bool) map[string]string {

	// Get annotations from Chart attributes
	ociAnnotations := generateChartOCIAnnotations(meta, test)

	// Copy Chart annotations
annotations:
	for chartAnnotationKey, chartAnnotationValue := range meta.Annotations {

		// Avoid overriding key properties
		for _, immutableOciKey := range immutableOciAnnotations {
			if immutableOciKey == chartAnnotationKey {
				continue annotations
			}
		}

		// Add chart annotation
		ociAnnotations[chartAnnotationKey] = chartAnnotationValue
	}

	return ociAnnotations
}

// getChartOCIAnnotations will generate OCI annotations from the provided chart
func generateChartOCIAnnotations(meta *common.Metadata, test bool) map[string]string {
	chartOCIAnnotations := map[string]string{}

	chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationDescription, meta.Description)
	chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationTitle, meta.Name)
	chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationVersion, meta.Version)
	chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationURL, meta.Home)

	if !test {
		chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationCreated, utils.Now().UTC().Format(time.RFC3339))
	}

	if len(meta.Sources) > 0 {
		chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationSource, meta.Sources[0])
	}

	if meta.Maintainers != nil && len(meta.Maintainers) > 0 {
		var maintainerSb strings.Builder

		for maintainerIdx, maintainer := range meta.Maintainers {

			if len(maintainer.Name) > 0 {
				maintainerSb.WriteString(maintainer.Name)
			}

			if len(maintainer.Email) > 0 {
				maintainerSb.WriteString(" (")
				maintainerSb.WriteString(maintainer.Email)
				maintainerSb.WriteString(")")
			}

			if maintainerIdx < len(meta.Maintainers)-1 {
				maintainerSb.WriteString(", ")
			}

		}

		chartOCIAnnotations = addToMap(chartOCIAnnotations, ocispec.AnnotationAuthors, maintainerSb.String())

	}

	return chartOCIAnnotations
}

// addToMap takes an existing map and adds an item if the value is not empty
func addToMap(inputMap map[string]string, newKey string, newValue string) map[string]string {

	// Add item to map if its
	if len(strings.TrimSpace(newValue)) > 0 {
		inputMap[newKey] = newValue
	}

	return inputMap

}

// NewClientTLS returns tls.Config appropriate for client auth.
func NewClientTLS(p RegistryParams) (*tls.Config, error) {
	config := tls.Config{
		InsecureSkipVerify: p.Insecure,
	}

	if p.CertFile != "" && p.KeyFile != "" {
		cert, err := CertFromFilePair(p.CertFile, p.KeyFile)
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{*cert}
	}

	if p.CAFile != "" {
		cp, err := CertPoolFromFile(p.CAFile)
		if err != nil {
			return nil, err
		}
		config.RootCAs = cp
	}

	return &config, nil
}

// CertPoolFromFile returns an x509.CertPool containing the certificates
// in the given PEM-encoded file.
// Returns an error if the file could not be read, a certificate could not
// be parsed, or if the file does not contain any certificates
func CertPoolFromFile(filename string) (*x509.CertPool, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Errorf("can't read CA file: %v", filename)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		return nil, errors.Errorf("failed to append certificates from file: %s", filename)
	}
	return cp, nil
}

// CertFromFilePair returns an tls.Certificate containing the
// certificates public/private key pair from a pair of given PEM-encoded files.
// Returns an error if the file could not be read, a certificate could not
// be parsed, or if the file does not contain any certificates
func CertFromFilePair(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrapf(err, "can't load key pair from cert %s and key %s", certFile, keyFile)
	}
	return &cert, err
}
