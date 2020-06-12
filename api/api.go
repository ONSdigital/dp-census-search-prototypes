package api

import (
	"context"

	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

var httpServer *server.Server

// SearchAPI manages searches across indices
type SearchAPI struct {
	defaultMaxResults int
	elasticsearch     Elasticsearcher
	router            *mux.Router
	datasetIndex      string
	postcodeIndex     string
	boundaryFileIndex string
}

// CreateAndInitialiseSearchAPI manages all the routes configured to API
func CreateAndInitialiseSearchAPI(ctx context.Context, bindAddr string, esAPI Elasticsearcher, defaultMaxResults int, datasetIndex, postcodeIndex, boundaryFileIndex string, errorChan chan error) {

	router := mux.NewRouter()
	routes(ctx,
		router,
		esAPI,
		defaultMaxResults,
		datasetIndex,
		postcodeIndex,
		boundaryFileIndex,
	)

	httpServer = server.New(bindAddr, router)

	// Disable this here to allow service to manage graceful shutdown of the entire app.
	httpServer.HandleOSSignals = false

	go func() {
		log.Event(ctx, "Starting api...", log.INFO)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Event(ctx, "api http server returned error", log.ERROR, log.Error(err))
			errorChan <- err
		}
	}()
}

func routes(ctx context.Context,
	router *mux.Router,
	elasticsearch Elasticsearcher,
	defaultMaxResults int,
	datasetIndex, postcodeIndex, boundaryFileIndex string) *SearchAPI {

	api := SearchAPI{
		defaultMaxResults: defaultMaxResults,
		elasticsearch:     elasticsearch,
		router:            router,
		datasetIndex:      datasetIndex,
		postcodeIndex:     postcodeIndex,
		boundaryFileIndex: boundaryFileIndex,
	}

	api.router.HandleFunc("/search/parent", api.postParentSearch).Methods("POST")
	api.router.HandleFunc("/search/parent/{id}", api.getParentSearch).Methods("GET")
	api.router.HandleFunc("/search/postcodes/{postcode}", api.getPostcodeSearch).Methods("GET")

	return &api
}

// Close represents the graceful shutting down of the http server
func Close(ctx context.Context) error {
	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}
	log.Event(ctx, "graceful shutdown of http server complete", log.INFO)
	return nil
}
