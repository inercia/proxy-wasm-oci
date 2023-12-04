package server

import (
	"context"
	"strconv"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/expvar"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/inercia/proxy-wasm-oci/pkg/config"
	"github.com/inercia/proxy-wasm-oci/pkg/registry"
)

type Server struct {
	log *zap.Logger
	*fiber.App

	settings       *config.GlobalSettings
	registryConfig *registry.Configuration
	downloads      singleflight.Group
}

// StartServer starts a Fiber server.
// The server will be gracefully shutdown when the context is canceled.
func NewServer(settings *config.GlobalSettings, l *zap.Logger, regCfg *registry.Configuration) (*Server, error) {
	log := l
	log.Info("Creating API server.")

	log.Info("API server: creating Fiber HTTP server.")
	fiberConfig := fiber.Config{
		Prefork:               false, // WARNING: do not fork or everything will break
		ServerHeader:          "Proxy-Wasm OCI",
		DisableStartupMessage: true,
		EnablePrintRoutes:     true,
	}

	appRoot := fiber.New(fiberConfig)
	res := &Server{
		log: log,
		App: appRoot,

		settings:       settings,
		registryConfig: regCfg,
		downloads:      singleflight.Group{},
	}

	appRoot.Use(fiberzap.New(fiberzap.Config{
		Logger: log,
	}))

	appRoot.Use(expvar.New())
	appRoot.Use(pprof.New())
	appRoot.Use(recover.New())
	appRoot.Use(etag.New())
	appRoot.Use(requestid.New())

	// server.Use(logger.HTTPAccessLogHandler(log))

	// We are ok with default CORS config for now
	// see https://docs.gofiber.io/api/middleware/cors/#default-config
	appRoot.Use(cors.New())

	appRootMain := appRoot.Group("/")
	RegisterWASMBridge(appRootMain, log, res)

	return res, nil
}

// StartServer starts a Fiber server.
// The server will be gracefully shutdown when the context is canceled.
func (server *Server) Start(ctx context.Context, port int) error {
	log := server.log.Named("start")

	go func() {
		// Wait until the context is cancelled, and then stop the application
		grace := DefMinGraceShutdownTimeout
		<-ctx.Done()

		log.Sugar().Infof("API server: starting graceful shutdown: waiting up to %s secs. for connections to finish...", grace)
		if err := server.ShutdownWithTimeout(grace); err != nil {
			log.Error("API server: shutdown error for API", zap.Error(err))
		} else {
			log.Info("API server: graceful shutdown of API completed. No API available from now on...")
		}
	}()

	log.Sugar().Infof("API server: listening on :%d", port)
	return server.Listen(getPortAsListenString(port))
}

func getPortAsListenString(port int) string {
	return ":" + strconv.Itoa(port)
}
