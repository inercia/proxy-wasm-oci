package server

import "time"

const (
	DefMinGraceShutdownTimeout = 2 * time.Second
)

const (
	// PathWASMDownload is the path where the Proxy-WASM binary can be downloaded.
	PathWASMDownload = "/api/v1/wasm/download"
)
