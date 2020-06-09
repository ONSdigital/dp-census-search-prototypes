package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
)

// ErrorUnexpectedStatusCode represents the error message to be returned when
// the status received from elastic is not as expected
var ErrorUnexpectedStatusCode = errors.New("unexpected status code from api")

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	clienter dphttp.Clienter
	url      string
}

// NewElasticSearchAPI creates an ElasticSearchAPI object
func NewElasticSearchAPI(clienter dphttp.Clienter, elasticSearchAPIURL string) *API {

	return &API{
		clienter: clienter,
		url:      elasticSearchAPIURL,
	}
}

// CreateSearchIndex creates a new index in elastic search
func (api *API) CreateSearchIndex(ctx context.Context, indexName string, mappingsFile string) (int, error) {
	path := api.url + "/" + indexName

	indexMappings, err := Asset(mappingsFile)
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "PUT", indexMappings)
	if err != nil {
		return status, err
	}

	return status, nil
}

// DeleteSearchIndex removes an index from elastic search
func (api *API) DeleteSearchIndex(ctx context.Context, indexName string) (int, error) {
	path := api.url + "/" + indexName

	_, status, err := api.CallElastic(ctx, path, "DELETE", nil)
	if err != nil {
		return status, err
	}

	return status, nil
}

// AddGeoLocation adds a document to an elasticsearch index
func (api *API) AddGeoLocation(ctx context.Context, indexName string, geoDoc *models.GeoDoc) (int, error) {
	if geoDoc == nil || geoDoc.Name == "" {
		return 0, errors.New("missing data")
	}

	log.Event(ctx, "adding geodoc", log.INFO, log.Data{"location": geoDoc.Name})

	path := api.url + "/" + indexName + "/_doc"

	bytes, err := json.Marshal(geoDoc)
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "POST", bytes)
	if err != nil {
		return status, err
	}

	return status, nil
}

// BulkRequest ...
func (api *API) BulkRequest(ctx context.Context, indexName string, documents []interface{}) (int, error) {
	path := api.url + "/_bulk"

	var bulk []byte

	for _, doc := range documents {

		b, err := json.Marshal(doc)
		if err != nil {
			return 0, err
		}

		bulk = append(bulk, []byte("{ \"index\": {\"_index\": \""+indexName+"\", \"_type\": \"_doc\"} }\n")...) // It may need an ID?
		bulk = append(bulk, b...)
		bulk = append(bulk, []byte("\n")...)
	}

	_, status, err := api.CallElastic(ctx, path, "POST", bulk)
	if err != nil {
		return status, err
	}

	return status, nil
}

// GetPostcodes searches index for resources containing postcode
func (api *API) GetPostcodes(ctx context.Context, indexName, postcode string) (*models.PostcodeResponse, int, error) {
	path := api.url + "/" + indexName + "/_search"

	logData := log.Data{"postcode": postcode, "path": path}
	log.Event(ctx, "get postcode", log.INFO, logData)

	body := models.PostcodeRequest{
		Query: models.PostcodeQuery{
			Distance: models.PostcodeTerm{
				Postcode: postcode,
			},
		},
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		log.Event(ctx, "unable to marshal elastic search query to bytes", log.ERROR, log.Error(err), logData)
		return nil, 0, errs.ErrMarshallingQuery
	}

	responseBody, status, err := api.CallElastic(ctx, path, "GET", bytes)
	if err != nil {
		return nil, status, err
	}

	response := &models.PostcodeResponse{}

	if err = json.Unmarshal(responseBody, response); err != nil {
		log.Event(ctx, "unable to unmarshal json body", log.ERROR, log.Error(err), logData)
		return nil, status, errs.ErrUnmarshallingJSON
	}

	return response, status, nil
}

// QueryGeoLocation ...
func (api *API) QueryGeoLocation(ctx context.Context, indexName string, geoLocation *models.GeoLocation, limit, offset int) (*models.GeoLocationResponse, int, error) {
	if geoLocation == nil || geoLocation.Type != "polygon" {
		return nil, 0, errors.New("missing data")
	}

	path := api.url + "/" + indexName + "/_search"

	query := buildGeoLocationQuery(*geoLocation)

	log.Event(ctx, "get documents based on geo polygon search", log.INFO, log.Data{"query": query, "path": path})

	bytes, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	responseBody, status, err := api.CallElastic(ctx, path, "POST", bytes)
	if err != nil {
		return nil, status, err
	}

	response := &models.GeoLocationResponse{}

	if err = json.Unmarshal(responseBody, response); err != nil {
		log.Event(ctx, "unable to unmarshal json body", log.ERROR, log.Error(err))
		return nil, status, errs.ErrUnmarshallingJSON
	}

	return response, status, nil
}

// CallElastic builds a request to elastic search based on the method, path and payload
func (api *API) CallElastic(ctx context.Context, path, method string, payload interface{}) ([]byte, int, error) {
	logData := log.Data{"url": path, "method": method}

	URL, err := url.Parse(path)
	if err != nil {
		log.Event(ctx, "failed to create url for elastic call", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}
	path = URL.String()
	logData["url"] = path

	var req *http.Request

	if payload != nil {
		req, err = http.NewRequest(method, path, bytes.NewReader(payload.([]byte)))
		req.Header.Add("Content-type", "application/json")
		logData["payload"] = string(payload.([]byte))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	// check req, above, didn't error
	if err != nil {
		log.Event(ctx, "failed to create request for call to elastic", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}

	resp, err := api.clienter.Do(ctx, req)
	if err != nil {
		log.Event(ctx, "failed to call elastic", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}
	defer resp.Body.Close()

	logData["http_code"] = resp.StatusCode

	jsonBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Event(ctx, "failed to read response body from call to elastic", log.ERROR, log.Error(err), logData)
		return nil, resp.StatusCode, err
	}
	logData["json_body"] = string(jsonBody)
	logData["status_code"] = resp.StatusCode

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		log.Event(ctx, "failed", log.ERROR, log.Error(ErrorUnexpectedStatusCode), logData)
		return nil, resp.StatusCode, ErrorUnexpectedStatusCode
	}

	return jsonBody, resp.StatusCode, nil
}

func buildGeoLocationQuery(geoLocation models.GeoLocation) models.GeoLocationRequest {
	return models.GeoLocationRequest{
		Query: models.GeoLocationQuery{
			Bool: models.BooleanObject{
				Must: models.MustObject{
					Match: models.MatchAll{},
				},
				Filter: models.GeoFilter{
					Shape: models.GeoShape{
						Location: models.GeoLocationObj{
							Shape:    geoLocation,
							Relation: "intersects",
						},
					},
				},
			},
		},
	}
}
