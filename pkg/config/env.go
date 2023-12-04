package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// defaultMaxHistory sets the maximum number of releases to 0: unlimited
const defaultMaxHistory = 10

// defaultBurstLimit sets the default client-side throttling limit
const defaultBurstLimit = 100

// GlobalSettings describes all of the environment settings.
type GlobalSettings struct {
	// Debug indicates whether or not Helm is running in Debug mode.
	Debug bool

	// RegistryConfigFilename is the path to the registry config file.
	RegistryConfigFilename string

	// BurstLimit is the default client-side throttling limit.
	BurstLimit int
}

func New() *GlobalSettings {
	env := &GlobalSettings{
		RegistryConfigFilename: envOr("PWO_REGISTRY_CONFIG", ConfigPath("registry/config.json")),
		BurstLimit:             envIntOr("PWO_BURST_LIMIT", defaultBurstLimit),
	}
	env.Debug, _ = strconv.ParseBool(os.Getenv("PWO_DEBUG"))

	return env
}

// AddFlags binds flags to the given flagset.
func (s *GlobalSettings) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.Debug, "debug", s.Debug, "enable verbose output")
	fs.StringVar(&s.RegistryConfigFilename, "registry-config", s.RegistryConfigFilename, "path to the registry config file")
	fs.IntVar(&s.BurstLimit, "burst-limit", s.BurstLimit, "client-side default throttling limit")
}

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}

func envBoolOr(name string, def bool) bool {
	if name == "" {
		return def
	}
	envVal := envOr(name, strconv.FormatBool(def))
	ret, err := strconv.ParseBool(envVal)
	if err != nil {
		return def
	}
	return ret
}

func envIntOr(name string, def int) int {
	if name == "" {
		return def
	}
	envVal := envOr(name, strconv.Itoa(def))
	ret, err := strconv.Atoi(envVal)
	if err != nil {
		return def
	}
	return ret
}

func envCSV(name string) (ls []string) {
	trimmed := strings.Trim(os.Getenv(name), ", ")
	if trimmed != "" {
		ls = strings.Split(trimmed, ",")
	}
	return
}

func (s *GlobalSettings) EnvVars() map[string]string {
	envvars := map[string]string{
		"PWO_BIN":             os.Args[0],
		"PWO_CACHE_HOME":      CachePath(""),
		"PWO_CONFIG_HOME":     ConfigPath(""),
		"PWO_DATA_HOME":       DataPath(""),
		"PWO_DEBUG":           fmt.Sprint(s.Debug),
		"PWO_REGISTRY_CONFIG": s.RegistryConfigFilename,
		"PWO_BURST_LIMIT":     strconv.Itoa(s.BurstLimit),
	}
	return envvars
}
