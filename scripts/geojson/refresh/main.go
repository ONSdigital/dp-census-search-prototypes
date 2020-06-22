package main

import (
	"context"
	"net/http"
	"os"

	"github.com/ONSdigital/dp-census-search-prototypes/config"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

const (
	mappingsFile = "geography-mappings.json"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "failed to retrieve configuration", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, cfg.ElasticSearchAPIURL)

	// delete existing elasticsearch index if already exists
	status, err := esAPI.DeleteSearchIndex(ctx, cfg.GeoFileIndex)
	if err != nil {
		if status != http.StatusNotFound {
			log.Event(ctx, "failed to delete index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "failed to delete index as index cannot be found, continuing", log.WARN, log.Error(err), log.Data{"status": status})
	}

	// create elasticsearch index with settings/mapping
	status, err = esAPI.CreateSearchIndex(ctx, cfg.GeoFileIndex, mappingsFile)
	if err != nil {
		log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
		os.Exit(1)
	}

	log.Event(ctx, "successfully refreshed "+cfg.GeoFileIndex+" index", log.INFO)
}
