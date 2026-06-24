package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name            string
		env             map[string]string
		wantHost        string
		wantPort        string
		wantDatabaseURL string
		wantListenAddr  string
	}{
		{
			name:            "uses defaults",
			wantHost:        defaultServerHost,
			wantPort:        defaultServerPort,
			wantDatabaseURL: defaultDatabaseURL,
			wantListenAddr:  "0.0.0.0:8080",
		},
		{
			name: "uses environment values",
			env: map[string]string{
				"SERVER_HOST":  "127.0.0.1",
				"SERVER_PORT":  "9090",
				"DATABASE_URL": "/tmp/co-review.db",
			},
			wantHost:        "127.0.0.1",
			wantPort:        "9090",
			wantDatabaseURL: "/tmp/co-review.db",
			wantListenAddr:  "127.0.0.1:9090",
		},
		{
			name: "blank environment values fall back to defaults",
			env: map[string]string{
				"SERVER_HOST":  " ",
				"SERVER_PORT":  " ",
				"DATABASE_URL": " ",
			},
			wantHost:        defaultServerHost,
			wantPort:        defaultServerPort,
			wantDatabaseURL: defaultDatabaseURL,
			wantListenAddr:  "0.0.0.0:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.Host != tt.wantHost {
				t.Fatalf("Host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.Port != tt.wantPort {
				t.Fatalf("Port = %q, want %q", cfg.Port, tt.wantPort)
			}
			if cfg.DatabaseURL != tt.wantDatabaseURL {
				t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.wantDatabaseURL)
			}
			if cfg.ListenAddr() != tt.wantListenAddr {
				t.Fatalf("ListenAddr() = %q, want %q", cfg.ListenAddr(), tt.wantListenAddr)
			}
		})
	}
}
