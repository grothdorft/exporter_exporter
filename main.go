// exporter_exporter is a reverse proxy for Prometheus exporters.
// It allows a single port to be exposed for multiple exporters,
// routing requests based on the 'module' query parameter.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
)

var (
	listenAddress = flag.String("web.listen-address", ":9999", "Address to listen on for web interface and telemetry.")
	configFile    = flag.String("config.file", "exporter_exporter.yaml", "Path to configuration file.")
	printVersion  = flag.Bool("version", false, "Print version information and exit.")
)

// Version information, set at build time via ldflags.
var (
	Version   = "dev"
	Revision  = "unknown"
	Branch    = "unknown"
	BuildDate = "unknown"
)

// Config holds the top-level configuration.
type Config struct {
	Modules map[string]ModuleConfig `yaml:"modules"`
}

// ModuleConfig defines how to proxy a single exporter module.
type ModuleConfig struct {
	Method  string            `yaml:"method"`
	HTTP    *HTTPConfig       `yaml:"http,omitempty"`
	Exec    *ExecConfig       `yaml:"exec,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// HTTPConfig holds configuration for HTTP-based proxying.
type HTTPConfig struct {
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
	Scheme  string `yaml:"scheme"`
	VerifySSL bool `yaml:"verify_ssl"`
}

// ExecConfig holds configuration for exec-based proxying.
type ExecConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Env     []string `yaml:"env"`
	Timeout string   `yaml:"timeout"`
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	cfg := &Config{}
	decoder := yaml.NewDecoder(f)
	decoder.SetStrict(true)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

func main() {
	flag.Parse()

	if *printVersion {
		fmt.Printf("exporter_exporter version=%s revision=%s branch=%s buildDate=%s\n",
			Version, Revision, Branch, BuildDate)
		os.Exit(0)
	}

	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	handler, err := NewProxyHandler(cfg)
	if err != nil {
		log.Fatalf("Error creating proxy handler: %v", err)
	}

	http.Handle("/proxy", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><title>Exporter Exporter</title></head>
<body><h1>Exporter Exporter</h1>
<p><a href="/proxy">Proxy</a></p>
</body></html>`)
	})

	log.Infof("Listening on %s", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
