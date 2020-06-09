package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	"github.com/ONSdigital/dp-census-search-prototypes/models"
	"github.com/ONSdigital/dp-census-search-prototypes/helpers"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

const (
	defaultLimit  = 50
	defaultOffset = 0
	defaultSegments = 30

	postcodeNotFound = "postcode not found"

	internalError         = "internal server error"
	exceedsDefaultMaximum = "the maximum offset has been reached, the offset cannot be more than"
	invalidDistanceParam  = "invalid distance value"
)

var (
	err        error
	reNotFound = regexp.MustCompile(`\bbody: (\w+ not found)[\n$]`)
)

func (api *SearchAPI) getPostcodeSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	postcode := vars["postcode"]

	p := strings.ReplaceAll(postcode, " ", "")
	lcPostcode := strings.ToLower(p)

	distance := r.FormValue("distance")
	requestedLimit := r.FormValue("limit")
	requestedOffset := r.FormValue("offset")

	logData := log.Data{
		"postcode":         lcPostcode,
		"postcode_raw":     postcode,
		"distance":         distance,
		"requested_limit":  requestedLimit,
		"requested_offset": requestedOffset,
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

	page := &models.PageVariables{
		DefaultMaxResults: api.defaultMaxResults,
		Limit:             limit,
		Offset:            offset,
	}

	distObj, err := models.ValidateDistance(distance); err != nil {
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
	response, _, err := api.elasticsearch.GetPostcodes(ctx, api.postcodeIndex, lcPostcode)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to search for postcode", log.ERROR, log.Error(err), logData)
		setErrorCode(w, err)
		return
	}

	// Calculate distance (in metres) based on distObj
	distance := distObj.CalculateDistanceInMetres(ctx)

	pcCoordinate:= helpers.Coordinate{
		Lat: response.Hits[0].Lat,
		Lon: response.Hits[0].Lon,
	}

	// build polygon from circle using long/lat of postcod and distance
	polygonShape,err := helpers.CircleToPolygon(pcCoordinate, distance, defaultSegments)
	if err != nil {
		setErrorCode(w, err)
	}

	geoLocation := models.GeoLocation{
		Type: "polygon", // make constant variable?
		Coordinates: polygonShape.Coordinates,
	}

	// TODO - Query Dataset Index with polygon search (intersect)
	response, _, err := api.elasticsearch.QueryGeoLocation(ctx, datasetIndex, page.Limit, page.Offset)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to query elastic search index", log.ERROR, log.Error(err), logData)
		return nil, err
	}

	searchResults := &models.SearchResults{
		Count:  response.Hits.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	for _, result := range response.Hits.HitList {
		result.Source.DimensionOptionURL = result.Source.URL
		result.Source.URL = ""

		result = getSnippets(ctx, result)

		doc := result.Source
		searchResults.Items = append(searchResults.Items, doc)
	}

	searchResults.Count = len(searchResults.Items)

	b, err := json.Marshal(searchResults)
	if err != nil {
		log.Event(ctx, "getPostcodeSearch endpoint: failed to marshal search resource into bytes", log.ERROR, log.Error(err), logData)
		return nil, errs.ErrInternalServer
	}

	setJSONContentType(w)
	_, err = w.Write(b)
	if err != nil {
		log.Event(ctx, "error writing response", log.ERROR, log.Error(err), logData)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Event(ctx, "getPostcodeSearch endpoint: successfully searched index", log.INFO, logData)
}

func getSnippets(ctx context.Context, result models.HitList) models.HitList {

	if len(result.Highlight.Code) > 0 {
		highlightedCode := result.Highlight.Code[0]
		var prevEnd int
		logData := log.Data{}
		for {
			start := prevEnd + strings.Index(highlightedCode, "\u0001S") + 1

			logData["start"] = start

			end := strings.Index(highlightedCode, "\u0001E")
			if end == -1 {
				break
			}
			logData["end"] = prevEnd + end - 2

			snippet := models.Snippet{
				Start: start,
				End:   prevEnd + end - 2,
			}

			prevEnd = snippet.End

			result.Source.Matches.Code = append(result.Source.Matches.Code, snippet)
			log.Event(ctx, "getPostcodeSearch endpoint: added code snippet", log.INFO, logData)

			highlightedCode = string(highlightedCode[end+2:])
		}
	}

	if len(result.Highlight.Label) > 0 {
		highlightedLabel := result.Highlight.Label[0]
		var prevEnd int
		logData := log.Data{}
		for {
			start := prevEnd + strings.Index(highlightedLabel, "\u0001S") + 1

			logData["start"] = start

			end := strings.Index(highlightedLabel, "\u0001E")
			if end == -1 {
				break
			}
			logData["end"] = prevEnd + end - 2

			snippet := models.Snippet{
				Start: start,
				End:   prevEnd + end - 2,
			}

			prevEnd = snippet.End

			result.Source.Matches.Label = append(result.Source.Matches.Label, snippet)
			log.Event(ctx, "getPostcodeSearch endpoint: added label snippet", log.INFO, logData)

			highlightedLabel = string(highlightedLabel[end+2:])
		}
	}

	return result
}

func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
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
	default:
		http.Error(w, internalError, http.StatusInternalServerError)
	}
}
