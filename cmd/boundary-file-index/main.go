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

const mappingsFile = "boundary-file-mappings.json"

func main() {
	ctx := context.Background()

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "failed to retrieve configuration", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, cfg.ElasticSearchAPIURL)

	// create elasticsearch index with settings/mapping
	status, err := esAPI.CreateSearchIndex(ctx, cfg.BoundaryFileIndex, mappingsFile)
	if err != nil {
		if status != http.StatusBadRequest {
			log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "index already exists", log.INFO)
	} else {
		log.Event(ctx, "successfully created index", log.INFO)
	}
}
