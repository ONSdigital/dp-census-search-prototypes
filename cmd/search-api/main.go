package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-census-search-prototypes/api"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

var (
	elasticSearchAPIURL = "http://localhost:9200"
	datasetIndex        = "test_geolocation"
	postcodeIndex       = "test_postcode"
	bindAddr            = ":10000"
	defaultMaxResults   = 1000
)

func main() {
	log.Namespace = "dp-search-api"
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Event(ctx, "application unexpectedly failed", log.ERROR, log.Error(err))
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, elasticSearchAPIURL)

	_, status, err := esAPI.CallElastic(ctx, elasticSearchAPIURL, "GET", nil)
	if err != nil {
		log.Event(ctx, "failed to start up, unable to connect to elastic search instance", log.ERROR, log.Error(err), log.Data{"http_status": status})
		return err
	}

	apiErrors := make(chan error, 1)

	api.CreateAndInitialiseSearchAPI(ctx, bindAddr, esAPI, defaultMaxResults, datasetIndex, postcodeIndex, apiErrors)

	// block until a fatal error occurs
	select {
	case err := <-apiErrors:
		log.Event(ctx, "api error received", log.ERROR, log.Error(err))
		return err
	case <-signals:
		log.Event(ctx, "os signal received", log.INFO)
	}

	return nil
}
