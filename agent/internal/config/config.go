package config

import (
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		URL   string `yaml:"url"`
		Token string `yaml:"token"`
	} `yaml:"server"`

	Buffer struct {
		Size int `yaml:"size"`
	} `yaml:"buffer"`

	Collectors struct {
		CPU struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"cpu"`

		GPU struct {
			Enabled  bool     `yaml:"enabled"`
			Interval string   `yaml:"interval"`
			Metrics  []string `yaml:"metrics"`
		} `yaml:"gpu"`

		Mem struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"mem"`

		Swap struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"swap"`

		Load struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"load"`

		Net struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"net"`

		ProcessStates struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"process_states"`

		Disk struct {
			Enabled      bool     `yaml:"enabled"`
			Interval     string   `yaml:"interval"`
			AutoDiscover bool     `yaml:"auto_discover"`
			ExcludeFS    []string `yaml:"exclude_fs"`
		} `yaml:"disk"`

		Temperatures struct {
			Enabled  bool     `yaml:"enabled"`
			Interval string   `yaml:"interval"`
			Sensors  []string `yaml:"sensors"`
		} `yaml:"temperatures"`

		Battery struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"battery"`

		Entropy struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"entropy"`

		TimeSync struct {
			Enabled  bool   `yaml:"enabled"`
			Interval string `yaml:"interval"`
		} `yaml:"time_sync"`
	} `yaml:"collectors"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Default configurations
	cfg := &Config{}
	cfg.Server.URL = "http://127.0.0.1:3000/api/metrics"
	cfg.Buffer.Size = 5000

	// Default enables
	cfg.Collectors.CPU.Enabled = true
	cfg.Collectors.CPU.Interval = "1m"

	cfg.Collectors.Mem.Enabled = true
	cfg.Collectors.Mem.Interval = "30s"

	cfg.Collectors.Net.Enabled = true
	cfg.Collectors.Net.Interval = "30s"

	cfg.Collectors.Disk.Enabled = true
	cfg.Collectors.Disk.Interval = "10m"
	cfg.Collectors.Disk.AutoDiscover = true
	cfg.Collectors.Disk.ExcludeFS = []string{"tmpfs", "devtmpfs", "squashfs", "iso9660"}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ParseInterval is a helper to convert strings like "1m" to time.Duration safely
func ParseInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Warn("Invalid duration format, using default (1m)", "input", s, "error", err)
		return time.Minute
	}
	return d
}
