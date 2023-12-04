package registry

import (
	"io"
)

type Configuration struct {
	// RegistryClient is a client for working with registries
	RegistryClient *Client

	Log func(string, ...interface{})
}

// RegistryLogin performs a registry login operation.
type RegistryLogin struct {
	RegistryParams
	cfg *Configuration
}

type RegistryLoginOpt func(*RegistryLogin)

// NewRegistryLogin creates a new RegistryLogin object with the given configuration.
func NewRegistryLogin(cfg *Configuration) *RegistryLogin {
	return &RegistryLogin{
		cfg: cfg,
	}
}

// Run executes the registry login operation
func (a *RegistryLogin) Run(out io.Writer, hostname string, username string, password string, opts ...any) error {
	for _, opt := range opts {
		switch r := opt.(type) {
		case RegistryLoginOpt:
			r(a)
		case RegistryParamsOpt:
			r(&a.RegistryParams)
		}
	}

	return a.cfg.RegistryClient.Login(
		hostname,
		LoginOptBasicAuth(username, password),
		LoginOptInsecure(a.Insecure),
		LoginOptTLSClientConfig(a.CertFile, a.KeyFile, a.CAFile))
}

////////////////////////////////////////////////////////////////////////////////////////

// RegistryLogout performs a registry login operation.
type RegistryLogout struct {
	cfg *Configuration
}

// NewRegistryLogout creates a new RegistryLogout object with the given configuration.
func NewRegistryLogout(cfg *Configuration) *RegistryLogout {
	return &RegistryLogout{
		cfg: cfg,
	}
}

// Run executes the registry logout operation
func (a *RegistryLogout) Run(out io.Writer, hostname string) error {
	return a.cfg.RegistryClient.Logout(hostname)
}
