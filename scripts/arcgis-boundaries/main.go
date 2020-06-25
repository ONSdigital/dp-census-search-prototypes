package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

const (
	elasticsearchAPIURL = "http://localhost:9200"
	geoFileIndex        = "test_arcgis"
	lsoaURL             = "https://services1.arcgis.com/ESMARspQHYMw9BZ9/arcgis/rest/services/LSOA_DEC_2011_EW_BFC/FeatureServer/0/query?where=1%3D1&outFields=*&outSR=4326&f=json"
	mappingsFile        = "geography-mappings.json"
)

var countCh = make(chan int)

func main() {
	ctx := context.Background()

	cli := dphttp.NewClient()
	esAPI := es.NewElasticSearchAPI(cli, elasticsearchAPIURL)

	// delete existing elasticsearch index if already exists
	status, err := esAPI.DeleteSearchIndex(ctx, geoFileIndex)
	if err != nil {
		if status != http.StatusNotFound {
			log.Event(ctx, "failed to delete index", log.ERROR, log.Error(err), log.Data{"status": status})
			os.Exit(1)
		}

		log.Event(ctx, "failed to delete index as index cannot be found, continuing", log.WARN, log.Error(err), log.Data{"status": status})
	}

	// create elasticsearch index with settings/mapping
	status, err = esAPI.CreateSearchIndex(ctx, geoFileIndex, mappingsFile)
	if err != nil {
		log.Event(ctx, "failed to create index", log.ERROR, log.Error(err), log.Data{"status": status})
		os.Exit(1)
	}

	go trackCounts(ctx)

	// make lsoa request
	docs, err := callArcGis(ctx, lsoaURL)
	if err != nil {
		log.Event(ctx, "failed to retrieve lsoa data from arcgis", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	// Iterate items for individual geo boundaries and store documents in elasticsearch
	if err = storeDocs(ctx, esAPI, geoFileIndex, docs); err != nil {
		log.Event(ctx, "failed to store lsoa data in elasticsearch", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "successfully got geo docs, see data", log.INFO)
}

func callArcGis(ctx context.Context, url string) (*geoDocs, error) {
	logData := log.Data{"url": url}

	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Event(ctx, "request to argis failed", log.ERROR, log.Error(err), logData)
		return nil, err
	}

	geoDocs, err := createGeoDoc(resp.Body)
	if err != nil {
		log.Event(ctx, "failed to create geoDocs", log.ERROR, log.Error(err), logData)
		return nil, err
	}

	return geoDocs, nil
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

type geoDocs struct {
	items []geoDoc `json:"features"`
}

type geoDoc struct {
	name        string        `json:"name"`
	code        string        `json:"code"`
	lsoa11nm    string        `json:"lsoa11nm"`
	lsoa11nmw   string        `json:"lsoa11nmw"`
	shapeArea   float64       `json:"shape_area"`
	shapeLength float64       `json:"shape_length"`
	location    geoLocation   `json:"location"`
	attributes  *attributeDoc `json:"attributes,omitempty`
	geometry    *geometryDoc  `json:"geometry",omitempty`
}

type geoLocation struct {
	gtype       string        `json:"type"`
	coordinates [][][]float64 `json:"coordinates"`
}

type attributeDoc struct {
	code        string  `json:"LSOA11CD"`
	lsoa11nm    string  `json:"LSOA11NM"`
	lsoa11nmw   string  `json:"LSOA11NMW"`
	shapeArea   float64 `json:"Shape__Area"`
	shapeLength float64 `json:"Shape__Length"`
}

type geometryDoc struct {
	coordinates [][][]float64 `json:"rings"`
}

func createGeoDoc(reader io.Reader) (*geoDocs, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errs.ErrUnableToReadMessage
	}

	var docs geoDocs

	err = json.Unmarshal(b, &docs)
	if err != nil {
		return nil, errs.ErrUnableToParseJSON
	}
	return &docs, nil
}

func storeDocs(ctx context.Context, esAPI *es.API, indexName string, docs *geoDocs) (err error) {
	count := 0
	var geoDocs []interface{}

	// Iterate through the records
	for _, doc := range docs.items {
		count++

		newDoc := &geoDoc{
			code:        doc.attributes.code,
			name:        doc.attributes.lsoa11nm,
			lsoa11nm:    doc.attributes.lsoa11nm,
			lsoa11nmw:   doc.attributes.lsoa11nmw,
			shapeArea:   doc.attributes.shapeArea,
			shapeLength: doc.attributes.shapeLength,
			location: geoLocation{
				gtype:       "polygon",
				coordinates: doc.geometry.coordinates,
			},
		}

		geoDocs = append(geoDocs, newDoc)

		if count == 500 {
			if _, err = esAPI.BulkRequest(ctx, indexName, geoDocs); err != nil {
				log.Event(ctx, "failed to upload document to index", log.ERROR, log.Error(err), log.Data{"count": count})
				return
			}

			countCh <- count

			count = 0
			geoDocs = nil
		}
	}

	// Capture last bulk
	if count != 0 {
		if _, err = esAPI.BulkRequest(ctx, indexName, geoDocs); err != nil {
			log.Event(ctx, "failed to upload document to index", log.ERROR, log.Error(err), log.Data{"count": count})
			return
		}

		countCh <- count

		count = 0
		geoDocs = nil
	}

	return
}
