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
	"github.com/ONSdigital/dp-census-search-prototypes/config"
	es "github.com/ONSdigital/dp-census-search-prototypes/elasticsearch"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

const (
	lsoaURL      = "https://services1.arcgis.com/ESMARspQHYMw9BZ9/arcgis/rest/services/LSOA_DEC_2011_EW_BFC/FeatureServer/0/query?where=1%3D1&outFields=*&outSR=4326&f=json"
	mappingsFile = "geography-mappings.json"
)

var countCh = make(chan int)

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

	go trackCounts(ctx)

	// make lsoa request
	docs, err := callArcGis(ctx, lsoaURL)
	if err != nil {
		log.Event(ctx, "failed to retrieve lsoa data from arcgis", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	// Iterate items for individual geo boundaries and store documents in elasticsearch
	if err = storeDocs(ctx, esAPI, cfg.GeoFileIndex, docs); err != nil {
		log.Event(ctx, "failed to store lsoa data in elasticsearch", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "successfully got geo docs, see data", log.INFO)
}

func callArcGis(ctx context.Context, url string) (*GeoDocs, error) {
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

type GeoDocs struct {
	Items []GeoDoc `json:"features"`
}

type GeoDoc struct {
	Name        string        `json:"name"`
	Code        string        `json:"code"`
	LSOA11NM    string        `json:"lsoa11nm"`
	LSOA11NMW   string        `json:"lsoa11nmw"`
	ShapeArea   float64       `json:"shape_area"`
	ShapeLength float64       `json:"shape_length"`
	Location    GeoLocation   `json:"location"`
	Attributes  *AttributeDoc `json:"attributes, omitempty`
	Geometry    *GeometryDoc  `json:"geometry", omitempty`
}

type GeoLocation struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

type AttributeDoc struct {
	Code        string  `json:"LSOA11CD"`
	LSOA11NM    string  `json:"LSOA11NM"`
	LSOA11NMW   string  `json:"LSOA11NMW"`
	ShapeArea   float64 `json:"Shape__Area"`
	ShapeLength float64 `json:"Shape__Length"`
}

type GeometryDoc struct {
	Coordinates [][][]float64 `json:"rings"`
}

func createGeoDoc(reader io.Reader) (*GeoDocs, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errs.ErrUnableToReadMessage
	}

	var geoDocs GeoDocs

	err = json.Unmarshal(b, &geoDocs)
	if err != nil {
		return nil, errs.ErrUnableToParseJSON
	}
	return &geoDocs, nil
}

func storeDocs(ctx context.Context, esAPI *es.API, indexName string, docs *GeoDocs) (err error) {
	count := 0
	var geoDocs []interface{}

	// Iterate through the records
	for _, doc := range docs.Items {
		count++

		newDoc := &GeoDoc{
			Code:        doc.Attributes.Code,
			Name:        doc.Attributes.LSOA11NM,
			LSOA11NM:    doc.Attributes.LSOA11NM,
			LSOA11NMW:   doc.Attributes.LSOA11NMW,
			ShapeArea:   doc.Attributes.ShapeArea,
			ShapeLength: doc.Attributes.ShapeLength,
			Location: GeoLocation{
				Type:        "polygon",
				Coordinates: doc.Geometry.Coordinates,
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
