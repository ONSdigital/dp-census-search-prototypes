package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

const (
	intersects = "intersects"

	boundaryFileNotFound = "boundary file does not exist by id"
)

func (api *SearchAPI) getParentSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	var err error

	id := vars["id"]

	requestedLimit := r.FormValue("limit")
	requestedOffset := r.FormValue("offset")

	logData := log.Data{
		"id":               id,
		"requested_limit":  requestedLimit,
		"requested_offset": requestedOffset,
	}

	log.Event(ctx, "getParentSearch endpoint: incoming request", log.INFO, logData)

	limit := defaultLimit
	if requestedLimit != "" {
		limit, err = strconv.Atoi(requestedLimit)
		if err != nil {
			log.Event(ctx, "getParentSearch endpoint: request limit parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, errs.ErrParsingQueryParameters)
			return
		}
	}

	offset := defaultOffset
	if requestedOffset != "" {
		offset, err = strconv.Atoi(requestedOffset)
		if err != nil {
			log.Event(ctx, "getParentSearch endpoint: request offset parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, errs.ErrParsingQueryParameters)
			return
		}
	}

	page := &models.PageVariables{
		DefaultMaxResults: api.defaultMaxResults,
		Limit:             limit,
		Offset:            offset,
	}

	if err = page.Validate(); err != nil {
		log.Event(ctx, "getParentSearch endpoint: validate pagination", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	logData["limit"] = page.Limit
	logData["offset"] = page.Offset

	log.Event(ctx, "getParentSearch endpoint: just before querying search index", log.INFO, logData)

	// lookup boundary file by id
	boundaryFileResponse, _, err := api.elasticsearch.GetBoundaryFile(ctx, api.boundaryFileIndex, id)
	if err != nil {
		log.Event(ctx, "getParentSearch endpoint: failed to search for boundary file", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	if len(boundaryFileResponse.Hits.Hits) < 1 {
		log.Event(ctx, "getParentSearch endpoint: failed to find boundary file", log.ERROR, log.Error(errs.ErrBoundaryFileNotFound), logData)
		setErrorCode(w, errs.ErrBoundaryFileNotFound)
		return
	}

	// retrieve location object from boundary file response
	geoLocation := &models.GeoLocation{
		Type:        boundaryFileResponse.Hits.Hits[0].Source.Location.Type,
		Coordinates: boundaryFileResponse.Hits.Hits[0].Source.Location.Coordinates,
	}

	// query dataset index with polygon search (intersect)
	response, _, err := api.elasticsearch.QueryGeoLocation(ctx, api.datasetIndex, geoLocation, page.Limit, page.Offset, intersects)
	if err != nil {
		log.Event(ctx, "getParentSearch endpoint: failed to query elastic search index", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	searchResults := &models.SearchResults{
		Count:  response.Hits.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	for _, result := range response.Hits.HitList {
		doc := result.Source
		searchResults.Items = append(searchResults.Items, doc)
	}

	b, err := json.Marshal(searchResults)
	if err != nil {
		log.Event(ctx, "getParentSearch endpoint: failed to marshal search resource into bytes", log.ERROR, log.Error(err), logData)
		setErrorCode(w, errs.ErrInternalServer)
	}

	setJSONContentType(w)
	_, err = w.Write(b)
	if err != nil {
		log.Event(ctx, "error writing response", log.ERROR, log.Error(err), logData)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Event(ctx, "getParentSearch endpoint: successfully searched index", log.INFO, logData)
}
