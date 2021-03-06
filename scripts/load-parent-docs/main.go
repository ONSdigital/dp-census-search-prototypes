package main

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

const (
	elasticsearchAPIURL = "http://localhost:9200"
	datasetIndex        = "test_parent"
	mappingsFile        = "parent-mappings.json"
)

var filename = "test-data/datasets"

func main() {
	ctx := context.Background()
	filename = filename + ".csv"

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, elasticsearchAPIURL)

	// delete existing elasticsearch index if already exists
	status, err := esAPI.DeleteSearchIndex(ctx, datasetIndex)
	if err != nil {
		if status != http.StatusNotFound {
			log.Event(ctx, "failed to delete index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "failed to delete index as index cannot be found, continuing", log.WARN, log.Error(err), log.Data{"status": status})
	}

	// create elasticsearch index with settings/mapping
	status, err = esAPI.CreateSearchIndex(ctx, datasetIndex, mappingsFile)
	if err != nil {
		log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
		os.Exit(1)
	}
	// upload geo locations from data/datasets-test.csv and manipulate data into models.GeoDoc
	if err = uploadDocs(ctx, esAPI, datasetIndex, filename); err != nil {
		log.Event(ctx, "failed to retrieve geo docs", log.ERROR, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "successfully loaded in geo docs", log.INFO)
}

func uploadDocs(ctx context.Context, esAPI *es.API, indexName, filename string) error {
	csvfile, err := os.Open(filename)
	if err != nil {
		log.Event(ctx, "failed to open the csv file", log.ERROR, log.Error(err))
		return err
	}

	// Parse the file
	r := csv.NewReader(csvfile)

	headerRow, err := r.Read()
	if err != nil {
		log.Event(ctx, "failed to read header row", log.ERROR, log.Error(err))
		return err
	}

	if err = check(headerRow); err != nil {
		log.Event(ctx, "header row missing expected headers", log.ERROR, log.Error(err))
		return err
	}

	count := 0

	// Iterate through the records
	for {
		count++
		// Read each record from csv
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Event(ctx, "failed to read row", log.ERROR, log.Error(err))
		}

		var geometry [][][]float64
		var coordinates [][]float64

		for i, value := range row {
			if i == 0 {
				log.Event(ctx, "first item in row - continuing with loop", log.INFO)
				continue
			}

			coordinate, err := convertCoordinate(value)
			if err != nil {
				return err
			}

			coordinates = append(coordinates, coordinate)
		}

		geometry = append(geometry, coordinates)

		geoDoc := &models.GeoDoc{
			Name: row[0],
			Location: models.GeoLocation{
				Type:        "Polygon",
				Coordinates: geometry,
			},
		}

		if _, err = esAPI.AddGeoLocation(ctx, indexName, geoDoc); err != nil {
			log.Event(ctx, "failed to upload document to index", log.ERROR, log.Error(err), log.Data{"count": count})
			return err
		}
	}

	return nil
}

const firstTerm = "name"

func check(headerRow []string) error {
	if len(headerRow) < 1 {
		return errors.New("empty header row")
	}

	if headerRow[0] != firstTerm {
		return errors.New("missing name header")
	}

	return nil
}

func convertCoordinate(coordinate string) (convertedLatLong []float64, err error) {
	latlong := strings.SplitN(coordinate, ",", 2)

	var lat, long float64
	lat, err = strconv.ParseFloat(latlong[0], 64)
	if err != nil {
		return
	}

	long, err = strconv.ParseFloat(latlong[1], 64)
	if err != nil {
		return
	}

	convertedLatLong = append(convertedLatLong, lat, long)

	return
}
