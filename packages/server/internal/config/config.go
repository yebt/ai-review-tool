package config

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
)

const (
	defaultServerHost  = "0.0.0.0"
	defaultServerPort  = "8080"
	defaultDatabaseURL = "./co_review.db"
)

// Config contains the runtime settings required to start the server.
type Config struct {
	Host        string
	Port        string
	DatabaseURL string
}

// Load reads server configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Host:        valueOrDefault(os.Getenv("SERVER_HOST"), defaultServerHost),
		Port:        valueOrDefault(os.Getenv("SERVER_PORT"), defaultServerPort),
		DatabaseURL: valueOrDefault(os.Getenv("DATABASE_URL"), defaultDatabaseURL),
	}
	vcfg := reflect.ValueOf(cfg)

	// VALIDATE CONFIGURATION FIELDS
	for i := 0; i < vcfg.NumField(); i++ {
		fValue := vcfg.Field(i)
		fType := vcfg.Type().Field(i)

		switch fValue.Kind() {
			case reflect.String:
				if strings.TrimSpace(fValue.String()) == "" {
					return Config{}, fmt.Errorf("%s must not be empty", fType.Name)
				}
			// default:
			// 	return Config{}, fmt.Errorf("%s has unsupported type %s", fType.Name, fValue.Kind())
		}
	}

	// if strings.TrimSpace(cfg.Host) == "" {
	// 	return Config{}, fmt.Errorf("SERVER_HOST must not be empty")
	// }
	// if strings.TrimSpace(cfg.Port) == "" {
	// 	return Config{}, fmt.Errorf("SERVER_PORT must not be empty")
	// }
	// if strings.TrimSpace(cfg.DatabaseURL) == "" {
	// 	return Config{}, fmt.Errorf("DATABASE_URL must not be empty")
	// }

	return cfg, nil
}

// ListenAddr returns the network address used by the HTTP server.
func (cfg Config) ListenAddr() string {
	return net.JoinHostPort(cfg.Host, cfg.Port)
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
