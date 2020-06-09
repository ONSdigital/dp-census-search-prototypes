package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-census-search-prototypes/api"
	"github.com/ONSdigital/dp-census-search-prototypes/config"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
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

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "failed to retrieve configuration", log.FATAL, log.Error(err))
		return err
	}

	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg})

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, cfg.ElasticSearchAPIURL)

	_, status, err := esAPI.CallElastic(ctx, cfg.ElasticSearchAPIURL, "GET", nil)
	if err != nil {
		log.Event(ctx, "failed to start up, unable to connect to elastic search instance", log.ERROR, log.Error(err), log.Data{"http_status": status})
		return err
	}

	apiErrors := make(chan error, 1)

	api.CreateAndInitialiseSearchAPI(ctx, cfg.BindAddr, esAPI, cfg.MaxSearchResultsOffset, cfg.DatasetIndex, cfg.PostcodeIndex, apiErrors)

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
