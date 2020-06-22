package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config is the filing resource handler config
type Config struct {
	BindAddr                  string `envconfig:"BIND_ADDR"                  json:"-"`
	BoundaryFileIndex         string `envconfig:"BOUNDARY_FILE_INDEX"`
	DatasetIndex              string `envconfig:"DATASET_INDEX"`
	ElasticSearchAPIURL       string `envconfig:"ELASTIC_SEARCH_URL"         json:"-"`
	GeoFileIndex              string `envconfig:"GEOFILE_INDEX"`
	MaxSearchResultsOffset    int    `envconfig:"MAX_SEARCH_RESULTS_OFFSET"`
	PostcodeIndex             string `envconfig:"POSTCODE_INDEX"`
	SignElasticsearchRequests bool   `envconfig:"SIGN_ELASTICSEARCH_REQUESTS"`
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                  ":10000",
		BoundaryFileIndex:         "test_boundary_files",
		DatasetIndex:              "test_geolocation",
		ElasticSearchAPIURL:       "http://localhost:9200",
		GeoFileIndex:              "test_geo",
		MaxSearchResultsOffset:    1000,
		PostcodeIndex:             "test_postcode",
		SignElasticsearchRequests: false,
	}

	return cfg, envconfig.Process("", cfg)
}
