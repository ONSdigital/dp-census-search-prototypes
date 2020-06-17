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

	"github.com/ONSdigital/dp-census-search-prototypes/config"
	"github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

var filename = "test-data/datasets"

const mappingsFile = "parent-mappings.json"

func main() {
	ctx := context.Background()
	filename = filename + ".csv"

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "failed to retrieve configuration", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, cfg.ElasticSearchAPIURL)

	// delete existing elasticsearch index if already exists
	status, err := esAPI.DeleteSearchIndex(ctx, cfg.DatasetIndex)
	if err != nil {
		if status != http.StatusNotFound {
			log.Event(ctx, "failed to delete index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "failed to delete index as index cannot be found, continuing", log.WARN, log.Error(err), log.Data{"status": status})
	}

	// create elasticsearch index with settings/mapping
	status, err = esAPI.CreateSearchIndex(ctx, cfg.DatasetIndex, mappingsFile)
	if err != nil {
		log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
		os.Exit(1)
	}
	// upload geo locations from data/datasets-test.csv and manipulate data into models.GeoDoc
	if err = uploadDocs(ctx, esAPI, cfg.DatasetIndex, filename); err != nil {
		log.Event(ctx, "failed to retrieve geo docs", log.ERROR, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "successfully loaded in geo docs", log.INFO)
}

func uploadDocs(ctx context.Context, esAPI *elasticsearch.API, indexName, filename string) error {
	csvfile, err := os.Open(filename)
	if err != nil {
		log.Event(ctx, "failed to open the csv file", log.ERROR, log.Error(err))
		return err
	}

	// Parse the file
	r := csv.NewReader(csvfile)
	//r := csv.NewReader(bufio.NewReader(csvfile))

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

		geoDoc := &models.GeoDoc{
			Name: row[0],
			Location: models.GeoLocation{
				Type: "Polygon",
			},
		}

		geoDoc.Location.Coordinates = append(geoDoc.Location.Coordinates, coordinates)

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
