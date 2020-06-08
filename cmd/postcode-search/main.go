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
	"time"

	"github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"

	"github.com/ONSdigital/dp-census-search-prototypes/models"
	"github.com/ONSdigital/log.go/log"
)

var (
	elasticSearchAPIURL = "http://localhost:9200"
	indexName           = "test_postcode"
	root                = "NSPL_FEB_2020_UK/Data/NSPL_FEB_2020_UK.csv"
	mappingsFile        = "postcode-mappings.json"

	countCh = make(chan int)
)

func main() {

	ctx := context.Background()

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, elasticSearchAPIURL)

	// delete existing elasticsearch index if already exists
	status, err := esAPI.DeleteSearchIndex(ctx, indexName)
	if err != nil {
		if status != http.StatusNotFound {
			log.Event(ctx, "failed to delete index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "failed to delete index as index cannot be found, continuing", log.WARN, log.Error(err), log.Data{"status": status})
	}

	// create elasticsearch index with settings/mapping
	status, err = esAPI.CreateSearchIndex(ctx, indexName, mappingsFile)
	if err != nil {
		log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
		os.Exit(1)
	}

	go trackCounts(ctx)

	if err = getPostcodeData(ctx, esAPI, root); err != nil {
		log.Event(ctx, "failed to get all postcode data into index", log.ERROR, log.Error(err))
		os.Exit(1)
	}
}

func getPostcodeData(ctx context.Context, esAPI *elasticsearch.API, filename string) error {
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

	var latcol, longcol int
	for i, value := range headerRow {
		if value == "lat" {
			latcol = i
			continue
		}

		if value == "long" {
			longcol = i
			continue
		}
	}

	if latcol == 0 || longcol == 0 {
		log.Event(ctx, "missing latitude or longitude header", log.INFO, log.Data{"lat_col": latcol, "long_col": longcol, "description": "lat and long should not be nil"})
		return errors.New("missing latitude or longitude header")
	}

	count := 0

	var postcodeDocs []interface{}

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
			break
		}

		lat, err := convertCoordinate(row[latcol])
		if err != nil {
			log.Event(ctx, "failed to convert latitude to float64", log.ERROR, log.Error(err))
			continue
		}

		long, err := convertCoordinate(row[longcol])
		if err != nil {
			log.Event(ctx, "failed to convert longitude to float64", log.ERROR, log.Error(err))
			continue
		}

		// remove whitspace from postcode
		postcode := strings.ReplaceAll(row[0], " ", "")

		lcPostcode := strings.ToLower(postcode)

		postcodeDoc := models.PostcodeDoc{
			Postcode:    lcPostcode,
			PostcodeRaw: row[0],
			Pin: models.PinObj{
				PinLocation: models.CoordinatePoint{
					Latitude:  lat,
					Longitude: long,
				},
			},
		}

		postcodeDocs = append(postcodeDocs, postcodeDoc)

		if count == 500 {
			if _, err = esAPI.BulkRequest(ctx, indexName, postcodeDocs); err != nil {
				log.Event(ctx, "failed to upload document to index", log.ERROR, log.Error(err), log.Data{"count": count})
				return err
			}

			countCh <- count

			count = 0
			postcodeDocs = nil
		}
	}

	// Capture last bulk
	if count != 0 {
		if _, err = esAPI.BulkRequest(ctx, indexName, postcodeDocs); err != nil {
			log.Event(ctx, "failed to upload document to index", log.ERROR, log.Error(err), log.Data{"count": count})
			return err
		}

		countCh <- count

		count = 0
		postcodeDocs = nil
	}

	return nil
}

func convertCoordinate(coordinate string) (convertedLatLong float64, err error) {
	convertedLatLong, err = strconv.ParseFloat(coordinate, 64)

	return
}

func trackCounts(ctx context.Context) {
	var (
		totalCounter = 0
	)

	t := time.NewTicker(5 * time.Second)

	for {
		select {
		case n := <-countCh:
			totalCounter += n
		case <-t.C:
			log.Event(ctx, "Total uploaded: "+strconv.Itoa(totalCounter), log.INFO)
		}
	}
}
