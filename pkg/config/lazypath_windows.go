//go:build windows

package config

import "os"

func dataHome() string { return configHome() }

func configHome() string { return os.Getenv("APPDATA") }

func cacheHome() string { return os.Getenv("TEMP") }
