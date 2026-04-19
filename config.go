package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"gopkg.in/yaml.v2"
)

// ModuleConfig holds configuration for a single exporter module.
type ModuleConfig struct {
	Method  string            `yaml:"method"`
	Timeout time.Duration     `yaml:"timeout"`
	Headers map[string]string `yaml:"headers"`

	HTTP *HTTPConfig `yaml:"http"`
	Exec *ExecConfig `yaml:"exec"`
}

// HTTPConfig holds configuration for HTTP-based proxying.
type HTTPConfig struct {
	Port        int    `yaml:"port"`
	Path        string `yaml:"path"`
	IPAddress   string `yaml:"ip_address"`
	Scheme      string `yaml:"scheme"`
	VerifySSL   bool   `yaml:"verify_ssl"`
}

// ExecConfig holds configuration for exec-based exporters.
type ExecConfig struct {
	Command []string `yaml:"command"`
	Env     []string `yaml:"env"`
}

// GlobalConfig holds global configuration options.
type GlobalConfig struct {
	TimeoutOffset time.Duration `yaml:"timeout_offset"`
}

// Config is the top-level configuration structure.
type Config struct {
	Global  GlobalConfig             `yaml:"global"`
	Modules map[string]*ModuleConfig `yaml:"modules"`
}

// UnmarshalYAML implements custom unmarshalling for ModuleConfig,
// setting sensible defaults before parsing user-provided values.
func (m *ModuleConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ModuleConfig
	// Set defaults
	m.Method = "http"
	m.Timeout = 10 * time.Second
	m.Headers = map[string]string{}

	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}

	switch m.Method {
	case "http", "exec":
		// valid
	default:
		return fmt.Errorf("unknown module method: %q", m.Method)
	}

	if m.Method == "http" && m.HTTP == nil {
		return fmt.Errorf("http method requires an http config block")
	}
	if m.Method == "exec" && m.Exec == nil {
		return fmt.Errorf("exec method requires an exec config block")
	}

	return nil
}

// UnmarshalYAML implements custom unmarshalling for HTTPConfig,
// setting defaults for optional fields.
func (h *HTTPConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain HTTPConfig
	h.Scheme = "http"
	h.Path = "/metrics"
	h.IPAddress = "127.0.0.1"
	h.VerifySSL = true

	if err := unmarshal((*plain)(h)); err != nil {
		return err
	}

	if h.Port == 0 {
		return fmt.Errorf("http config requires a port")
	}

	return nil
}

// validModuleName matches valid module name identifiers.
var validModuleName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate checks the overall configuration for correctness.
func (c *Config) Validate() error {
	for name := range c.Modules {
		if !validModuleName.MatchString(name) {
			return fmt.Errorf("invalid module name %q: must match [a-zA-Z0-9_-]+", name)
		}
	}
	return nil
}

// ProxyURL builds the target URL for an HTTP module.
func (m *ModuleConfig) ProxyURL() string {
	if m.HTTP == nil {
		return ""
	}
	return fmt.Sprintf("%s://%s:%d%s", m.HTTP.Scheme, m.HTTP.IPAddress, m.HTTP.Port, m.HTTP.Path)
}

// RoundTrip adds configured headers to outgoing requests.
func (m *ModuleConfig) RoundTrip(rt http.RoundTripper) http.RoundTripper {
	return headerRoundTripper{headers: m.Headers, wrapped: rt}
}

type headerRoundTripper struct {
	headers map[string]string
	wrapped http.RoundTripper
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return h.wrapped.RoundTrip(req)
}

// loadConfigFromFile reads and parses a YAML config file.
func loadConfigFromFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.UnmarshalStrict(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}
