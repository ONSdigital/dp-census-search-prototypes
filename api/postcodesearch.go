package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	"github.com/ONSdigital/dp-census-search-prototypes/helpers"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

const (
	defaultLimit    = 50
	defaultOffset   = 0
	defaultSegments = 30
	defaultRelation = "within"

	postcodeNotFound = "postcode not found"

	internalError         = "internal server error"
	exceedsDefaultMaximum = "the maximum offset has been reached, the offset cannot be more than"
	invalidDistanceParam  = "invalid distance value"
	invalidRelationParam  = "incorrect relation value"
)

func (api *SearchAPI) getPostcodeSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	setAccessControl(w, http.MethodGet)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var err error

	postcode := vars["postcode"]

	p := strings.ReplaceAll(postcode, " ", "")
	lcPostcode := strings.ToLower(p)

	distance := r.FormValue("distance")
	requestedLimit := r.FormValue("limit")
	requestedOffset := r.FormValue("offset")
	requestedRelation := r.FormValue("relation")

	logData := log.Data{
		"postcode":           lcPostcode,
		"postcode_raw":       postcode,
		"distance":           distance,
		"requested_limit":    requestedLimit,
		"requested_offset":   requestedOffset,
		"requested_relation": requestedRelation,
	}

	log.Event(ctx, "getPostcodeSearch endpoint: incoming request", log.INFO, logData)

	limit := defaultLimit
	if requestedLimit != "" {
		limit, err = strconv.Atoi(requestedLimit)
		if err != nil {
			log.Event(ctx, "getPostcodeSearch endpoint: request limit parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, errs.ErrParsingQueryParameters)
			return
		}
	}

	offset := defaultOffset
	if requestedOffset != "" {
		offset, err = strconv.Atoi(requestedOffset)
		if err != nil {
			log.Event(ctx, "getPostcodeSearch endpoint: request offset parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, errs.ErrParsingQueryParameters)
			return
		}
	}

	relation := defaultRelation
	if requestedRelation != "" {
		relation, err = models.ValidateRelation(requestedRelation)
		if err != nil {
			log.Event(ctx, "getPostcodeSearch endpoint: request relation parameter error", log.ERROR, log.Error(err), logData)
			setErrorCode(w, err)
			return
		}
	}

	page := &models.PageVariables{
		DefaultMaxResults: api.defaultMaxResults,
		Limit:             limit,
		Offset:            offset,
	}

	distObj, err := models.ValidateDistance(distance)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: validate query param, distance", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	if err = page.Validate(); err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: validate pagination", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	logData["limit"] = page.Limit
	logData["offset"] = page.Offset

	log.Event(ctx, "getPostcodeSearch endpoint: just before querying search index", log.INFO, logData)

	// lookup postcode
	postcodeResponse, _, err := api.elasticsearch.GetPostcodes(ctx, api.postcodeIndex, lcPostcode)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to search for postcode", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	if len(postcodeResponse.Hits.Hits) < 1 {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to find postcode", log.ERROR, log.Error(errs.ErrPostcodeNotFound), logData)
		setErrorCode(w, errs.ErrPostcodeNotFound)
		return
	}

	// calculate distance (in metres) based on distObj
	dist := distObj.CalculateDistanceInMetres(ctx)

	pcCoordinate := helpers.Coordinate{
		Lat: postcodeResponse.Hits.Hits[0].Source.Pin.Location.Lat,
		Lon: postcodeResponse.Hits.Hits[0].Source.Pin.Location.Lon,
	}

	// build polygon from circle using long/lat of postcod and distance
	polygonShape, err := helpers.CircleToPolygon(pcCoordinate, dist, defaultSegments)
	if err != nil {
		setErrorCode(w, err)
	}

	var coordinates [][][]float64
	geoLocation := &models.GeoLocation{
		Type:        "polygon", // TODO make constant variable?
		Coordinates: append(coordinates, polygonShape.Coordinates),
	}

	// query dataset index with polygon search (intersect)
	response, _, err := api.elasticsearch.QueryGeoLocation(ctx, api.datasetIndex, geoLocation, page.Limit, page.Offset, relation)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to query elastic search index", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	searchResults := &models.SearchResults{
		TotalCount: response.Hits.Total,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}

	for _, result := range response.Hits.HitList {
		doc := result.Source
		searchResults.Items = append(searchResults.Items, doc)
	}

	searchResults.Count = len(searchResults.Items)

	b, err := json.Marshal(searchResults)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to marshal search resource into bytes", log.ERROR, log.Error(err), logData)
		setErrorCode(w, errs.ErrInternalServer)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		log.Event(ctx, "error writing response", log.ERROR, log.Error(err), logData)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Event(ctx, "getPostcodeSearch endpoint: successfully searched index", log.INFO, logData)
}

func setErrorCode(w http.ResponseWriter, err error) {

	switch {
	case errs.NotFoundMap[err]:
		http.Error(w, err.Error(), http.StatusNotFound)
	case errs.BadRequestMap[err]:
		http.Error(w, err.Error(), http.StatusBadRequest)
	case strings.Contains(err.Error(), exceedsDefaultMaximum):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case strings.Contains(err.Error(), invalidDistanceParam):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case strings.Contains(err.Error(), invalidRelationParam):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, internalError, http.StatusInternalServerError)
	}
}
