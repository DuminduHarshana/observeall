package config

// Project holds the ObserveAll project configuration (observeall.yaml).
type Project struct {
	Name           string                   `yaml:"name"`
	Environment    string                   `yaml:"environment"`
	Language       string                   `yaml:"language"`
	Backends       []string                 `yaml:"backends"`
	Signals        []string                 `yaml:"signals"`
	Sampling       string                   `yaml:"sampling"`
	BackendConfigs map[string]BackendConfig  `yaml:"backend_configs,omitempty"`
}

// BackendConfig holds credentials/endpoints for a single backend.
type BackendConfig struct {
	Type     string `yaml:"type"`
	Endpoint string `yaml:"endpoint,omitempty"`
	APIKey   string `yaml:"api_key,omitempty"`
	Site     string `yaml:"site,omitempty"`
}

