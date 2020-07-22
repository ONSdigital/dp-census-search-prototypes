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

func (api *SearchAPI) getPlaceNameSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	setAccessControl(w, http.MethodGet)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var err error

	placename := vars["name"]

	requestedLimit := r.FormValue("limit")
	requestedOffset := r.FormValue("offset")

	logData := log.Data{
		"place_name":       placename,
		"requested_limit":  requestedLimit,
		"requested_offset": requestedOffset,
	}

	limit := defaultLimit
	if requestedLimit != "" {
		limit, err = strconv.Atoi(requestedLimit)
		if err != nil {
			log.Event(ctx, "getPlaceNameSearch endpoint: request limit parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, errs.ErrParsingQueryParameters)
			return
		}
	}

	offset := defaultOffset
	if requestedOffset != "" {
		offset, err = strconv.Atoi(requestedOffset)
		if err != nil {
			log.Event(ctx, "getPlaceNameSearch endpoint: request offset parameter error", log.ERROR, log.Error(err), logData)
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
		log.Event(ctx, "getPlaceNameSearch endpoint: validate pagination", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	logData["limit"] = page.Limit
	logData["offset"] = page.Offset

	log.Event(ctx, "getPlaceNameSearch endpoint: just before querying search index", log.INFO, logData)

	// build dataset search query
	query := buildSearchQuery(placename, limit, offset)

	// query geographical areas index with text search
	response, _, err := api.elasticsearch.GetBoundaryFiles(ctx, api.datasetIndex, query)
	if err != nil {
		log.Event(ctx, "getParentSearch endpoint: failed to query elastic search index", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	searchResults := &models.SearchResultsWithLocation{
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

	_, err = w.Write(b)
	if err != nil {
		log.Event(ctx, "error writing response", log.ERROR, log.Error(err), logData)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Event(ctx, "getParentSearch endpoint: successfully searched index", log.INFO, logData)
}

func buildSearchQuery(placename string, limit, offset int) interface{} {

	name := make(map[string]string)
	name["name"] = placename

	nameMatch := models.Match{
		Match: name,
	}

	scores := models.Scores{
		Score: models.Score{
			Order: "desc",
		},
	}

	listOfScores := []models.Scores{}
	listOfScores = append(listOfScores, scores)

	query := &models.Body{
		From: offset,
		Size: limit,
		Query: models.Query{
			Bool: models.Bool{
				Should: []models.Match{
					nameMatch,
				},
			},
		},
		Sort:      listOfScores,
		TotalHits: true,
	}

	return query
}
