package config

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/JWhist/jwconfig"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Firehose FirehoseConfig `yaml:"firehose"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port            string        `yaml:"port" default:"8080"`
	Host            string        `yaml:"host" default:"localhost"`
	MetricsPort     string        `yaml:"metrics_port" default:"9090"`
	MetricsHost     string        `yaml:"metrics_host" default:"localhost"`
	MaxConnections  int           `yaml:"max_connections" default:"1000"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"10s"`
	CORS            CORSConfig    `yaml:"cors"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowAllOrigins bool     `yaml:"allow_all_origins" default:"true"`
	AllowedOrigins  []string `yaml:"allowed_origins"`
	AllowedMethods  []string `yaml:"allowed_methods" default:"[\"GET\", \"POST\", \"PUT\", \"DELETE\", \"OPTIONS\"]"`
	AllowedHeaders  []string `yaml:"allowed_headers" default:"[\"*\"]"`
}

// FirehoseConfig contains AT Protocol firehose configuration
type FirehoseConfig struct {
	URL            string        `yaml:"url" default:"wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"`
	ReconnectDelay time.Duration `yaml:"reconnect_delay" default:"5s"`
	MaxReconnects  int           `yaml:"max_reconnects" default:"10"`
	ReadTimeout    time.Duration `yaml:"read_timeout" default:"60s"`
	WriteTimeout   time.Duration `yaml:"write_timeout" default:"10s"`
	PingInterval   time.Duration `yaml:"ping_interval" default:"30s"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level" default:"info"`
	Format     string `yaml:"format" default:"text"`
	Output     string `yaml:"output" default:"stdout"`
	Structured bool   `yaml:"structured" default:"false"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	var cfg Config

	if err := jwconfig.Load(filename, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadConfigWithDefaults loads configuration with defaults applied
func LoadConfigWithDefaults(filename string) (*Config, error) {
	cfg := jwconfig.LoadFile[Config](filename)

	// Validate and apply defaults
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDefaultConfig returns a configuration with all default values
func GetDefaultConfig() *Config {
	cfg := &Config{}
	err := cfg.Validate() // This will apply defaults
	if err != nil {
		// Handle error if needed, for now just print
		fmt.Println("Error validating default config:", err)
		panic(err)
	}
	return cfg
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}

	// Validate port is a valid number
	if _, err := strconv.Atoi(c.Server.Port); err != nil {
		return fmt.Errorf("invalid port number: %s", c.Server.Port)
	}

	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}

	if c.Server.MaxConnections <= 0 {
		c.Server.MaxConnections = 1000
	}

	if c.Server.ShutdownTimeout <= 0 {
		c.Server.ShutdownTimeout = 10 * time.Second
	}

	// Firehose validation
	if c.Firehose.URL == "" {
		c.Firehose.URL = "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"
	}

	// Validate firehose URL
	if _, err := url.Parse(c.Firehose.URL); err != nil {
		return fmt.Errorf("invalid firehose URL: %s", c.Firehose.URL)
	}

	if c.Firehose.ReconnectDelay <= 0 {
		c.Firehose.ReconnectDelay = 5 * time.Second
	}

	if c.Firehose.MaxReconnects <= 0 {
		c.Firehose.MaxReconnects = 10
	}

	// Logging validation
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	} else if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error", c.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"text": true, "json": true,
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	} else if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s, must be one of: text, json", c.Logging.Format)
	}

	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}

	return nil
}

// GetListenAddress returns the formatted listen address
func (c *Config) GetListenAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}

// GetBaseURL returns the base URL for the server
func (c *Config) GetBaseURL() string {
	if c.Server.Host == "0.0.0.0" {
		return "http://localhost:" + c.Server.Port
	}
	return "http://" + c.Server.Host + ":" + c.Server.Port
}
