package server

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
	"go.uber.org/zap"
)

// RegisterWASMBridge func for common paths (unauthenticated).
func RegisterWASMBridge(a fiber.Router, log *zap.Logger, server *Server) {

	a.Get(PathWASMDownload, func(c *fiber.Ctx) error {
		log.Info("Received request to download WASM extension")

		m := c.Queries()
		ref, ok := m["ref"]
		if !ok {
			log.Error("no 'ref' found in request")
			return fiber.ErrBadRequest
		}
		log := log.With(zap.String("ref", ref))

		log.Info("Valid request")

		regCfg := registry.RegistryParams{
			// TODO: maybe we should set some things here from query parameters...
		}

		// create a temp directory to download the extension
		tempDir, err := os.MkdirTemp("", "pwo-download-*")
		if err != nil {
			log.Error("could not create temporary directory", zap.Error(err))
			return fiber.ErrInternalServerError
		}
		// TODO: we should remove temporary directories

		fileName, err := DownloadWASMExtension(log, server, ref, tempDir, regCfg)
		if err != nil {
			log.Error("error downloading WASM extension", zap.Error(err))
			return fiber.ErrInternalServerError
		}

		return c.SendFile(fileName, false)
	})
}
