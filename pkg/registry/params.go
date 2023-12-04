package registry

type RegistryParams struct {
	CertFile  string
	KeyFile   string
	CAFile    string
	Insecure  bool
	PlainHTTP bool
}

// PushOpt is a type of function that sets options for a push action.
type RegistryParamsOpt func(*RegistryParams)

// WithCertFile specifies the path to the certificate file to use for TLS.
func WithCertFile(certFile string) RegistryParamsOpt {
	return func(r *RegistryParams) {
		r.CertFile = certFile
	}
}

// WithKeyFile specifies the path to the key file to use for TLS.
func WithKeyFile(keyFile string) RegistryParamsOpt {
	return func(r *RegistryParams) {
		r.KeyFile = keyFile
	}
}

// WithCAFile specifies the path to the CA file to use for TLS.
func WithCAFile(caFile string) RegistryParamsOpt {
	return func(r *RegistryParams) {
		r.CAFile = caFile
	}
}

// WithTLSClientConfig sets the certFile, keyFile, and caFile fields on the push configuration object.
func WithTLSClientConfig(certFile, keyFile, caFile string) RegistryParamsOpt {
	return func(p *RegistryParams) {
		p.CertFile = certFile
		p.KeyFile = keyFile
		p.CAFile = caFile
	}
}

// WithKeyFile specifies whether to very certificates when communicating.
func WithInsecure(insecure bool) RegistryParamsOpt {
	return func(r *RegistryParams) {
		r.Insecure = insecure
	}
}

// WithPlainHTTP configures the use of plain HTTP connections.
func WithPlainHTTP(plainHTTP bool) RegistryParamsOpt {
	return func(p *RegistryParams) {
		p.PlainHTTP = plainHTTP
	}
}
