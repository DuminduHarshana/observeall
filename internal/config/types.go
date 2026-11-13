package config

// Project holds the ObserveAll project configuration (observeall.yaml).
type Project struct {
	Name           string                  `yaml:"name" json:"name"`
	Environment    string                  `yaml:"environment" json:"environment"`
	Language       string                  `yaml:"language" json:"language"`
	Backends       []string                `yaml:"backends" json:"backends"`
	Signals        []string                `yaml:"signals" json:"signals"`
	Sampling       string                  `yaml:"sampling" json:"sampling"`
	BackendConfigs map[string]BackendConfig `yaml:"backend_configs,omitempty" json:"backend_configs,omitempty"`

	// PII scrubbing — when true, injects transform/pii_scrubber into the pipeline.
	PIIScrubbing   bool   `yaml:"pii_scrubbing,omitempty" json:"pii_scrubbing,omitempty"`
	PIIReplacement string `yaml:"pii_replacement,omitempty" json:"pii_replacement,omitempty"`
}

// BackendConfig holds credentials/endpoints for a single backend.
type BackendConfig struct {
	Type     string `yaml:"type" json:"type"`
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	APIKey   string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	Site     string `yaml:"site,omitempty" json:"site,omitempty"`
}

