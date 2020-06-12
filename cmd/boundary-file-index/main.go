package main

import (
	"context"
	"net/http"
	"os"

	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

var (
	elasticSearchAPIURL = "http://localhost:9200"
	indexName           = "test_boundary_files"
	mappingsFile        = "boundary-file-mappings.json"
)

func main() {
	ctx := context.Background()

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, elasticSearchAPIURL)

	// create elasticsearch index with settings/mapping
	status, err := esAPI.CreateSearchIndex(ctx, indexName, mappingsFile)
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
