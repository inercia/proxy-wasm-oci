package registry

import "github.com/spf13/pflag"

func AddRegistryParamsFlags(f *pflag.FlagSet, r *RegistryParams) {
	f.StringVar(&r.CertFile, "cert-file", "", "identify registry client using this SSL certificate file")
	f.StringVar(&r.KeyFile, "key-file", "", "identify registry client using this SSL key file")
	f.StringVar(&r.CAFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")
	f.BoolVar(&r.Insecure, "insecure-skip-tls-verify", false, "skip tls certificate checks for the chart upload")
	f.BoolVar(&r.PlainHTTP, "plain-http", false, "use insecure HTTP connections for the chart upload")
}
