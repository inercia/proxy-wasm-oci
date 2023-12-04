package cmd

import (
	"context"
	"io"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/inercia/proxy-wasm-oci/pkg/registry"
	"github.com/inercia/proxy-wasm-oci/pkg/server"
)

const DefListenPort = 15111

const serveDesc = `
Serve Proxy-Wasm extensions from an OCI registry through HTTP.

Example:

  $ pwo serve --port 17000
`

func newServeCmd(cfg *registry.Configuration, l *zap.Logger, out io.Writer) *cobra.Command {
	log := l.Named("server")
	listenPort := 0

	cmd := &cobra.Command{
		Use:     "serve [remote]",
		Short:   "serve Proxy-Wasm extensionss from OCI registries through HTTP",
		Aliases: []string{"server"},
		Long:    downloadDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			ctx := context.Background()

			srv, err := server.NewServer(settings, log, cfg)
			if err != nil {
				return err
			}

			log.Info("Starting API server...")
			go func() {
				defer wg.Done()
				if err = srv.Start(ctx, listenPort); err != nil {
					log.Error("Error running API server", zap.Error(err))
				}
			}()
			wg.Add(1)

			wg.Wait()

			return nil
		},
	}

	f := cmd.Flags()
	f.IntVar(&listenPort, "port", DefListenPort, "port to listen at, as PORT")

	return cmd
}
