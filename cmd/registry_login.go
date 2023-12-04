package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/moby/term"
	"github.com/spf13/cobra"

	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

const registryLoginDesc = `
Authenticate to a remote registry.
`

type registryLoginOptions struct {
	username             string
	password             string
	passwordFromStdinOpt bool
	certFile             string
	keyFile              string
	caFile               string
	insecure             bool
}

func newRegistryLoginCmd(cfg *registry.Configuration, out io.Writer) *cobra.Command {
	o := &registryLoginOptions{}

	cmd := &cobra.Command{
		Use:   "login [host]",
		Short: "login to a registry",
		Long:  registryLoginDesc,
		Args:  MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := args[0]

			username, password, err := getUsernamePassword(o.username, o.password, o.passwordFromStdinOpt)
			if err != nil {
				return err
			}

			return registry.NewRegistryLogin(cfg).Run(out, hostname, username, password,
				registry.WithCertFile(o.certFile),
				registry.WithKeyFile(o.keyFile),
				registry.WithCAFile(o.caFile),
				registry.WithInsecure(o.insecure))
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.username, "username", "u", "", "registry username")
	f.StringVarP(&o.password, "password", "p", "", "registry password or identity token")
	f.BoolVarP(&o.passwordFromStdinOpt, "password-stdin", "", false, "read password or identity token from stdin")

	f.BoolVarP(&o.insecure, "insecure", "", false, "allow connections to TLS registry without certs")
	f.StringVar(&o.certFile, "cert-file", "", "identify registry client using this SSL certificate file")
	f.StringVar(&o.keyFile, "key-file", "", "identify registry client using this SSL key file")
	f.StringVar(&o.caFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")

	return cmd
}

// Adapted from https://github.com/oras-project/oras
func getUsernamePassword(usernameOpt string, passwordOpt string, passwordFromStdinOpt bool) (string, string, error) {
	var err error
	username := usernameOpt
	password := passwordOpt

	if passwordFromStdinOpt {
		passwordFromStdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", "", err
		}
		password = strings.TrimSuffix(string(passwordFromStdin), "\n")
		password = strings.TrimSuffix(password, "\r")
	} else if password == "" {
		if username == "" {
			username, err = readLine("Username: ", false)
			if err != nil {
				return "", "", err
			}
			username = strings.TrimSpace(username)
		}
		if username == "" {
			password, err = readLine("Token: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", errors.New("token required")
			}
		} else {
			password, err = readLine("Password: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", errors.New("password required")
			}
		}
	} else {
		warning("Using --password via the CLI is insecure. Use --password-stdin.")
	}

	return username, password, nil
}

// Copied/adapted from https://github.com/oras-project/oras
func readLine(prompt string, silent bool) (string, error) {
	fmt.Print(prompt)
	if silent {
		fd := os.Stdin.Fd()
		state, err := term.SaveState(fd)
		if err != nil {
			return "", err
		}
		term.DisableEcho(fd, state)
		defer term.RestoreTerminal(fd, state)
	}

	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if silent {
		fmt.Println()
	}

	return string(line), nil
}
