package downloader

import (
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
)

func AddDownloadFlags(f *pflag.FlagSet, c *CommonPullOptions) {
	f.StringVar(&c.Version, "version", "", "specify a version constraint for the Proxy-WASM extension version to use. This constraint can be a specific tag (e.g. 1.1.1) or it may reference a valid range (e.g. ^2.0.0). If this is not specified, the latest version is used")
	f.BoolVar(&c.Verify, "verify", false, "verify the package before using it")
	f.StringVar(&c.Username, "username", "", "repository username where to locate the requested Proxy-WASM Extension")
	f.StringVar(&c.Password, "password", "", "repository password where to locate the requested Proxy-WASM Extension")
	f.BoolVar(&c.PassCredentialsAll, "pass-credentials", false, "pass credentials to all domains")
}

// defaultKeyring returns the expanded path to the default keyring.
func defaultKeyring() string {
	if v, ok := os.LookupEnv("GNUPGHOME"); ok {
		return filepath.Join(v, "pubring.gpg")
	}
	return filepath.Join(homedir.HomeDir(), ".gnupg", "pubring.gpg")
}
