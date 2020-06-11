package models

import (
	"errors"
	"strings"
)

type GeoDoc struct {
	Name     string      `json:"name"`
	Location GeoLocation `json:"location"`
}

type GeoLocation struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// ------------------------------------------------------------------------

type GeoLocationRequest struct {
	Query GeoLocationQuery `json:"query"`
}

type GeoLocationQuery struct {
	Bool BooleanObject `json:"bool"`
}

type BooleanObject struct {
	Must   MustObject `json:"must"`
	Filter GeoFilter  `json:"filter"`
}

type MustObject struct {
	Match MatchAll `json:"match_all"`
}

type MatchAll struct{}

type GeoFilter struct {
	Shape GeoShape `json:"geo_shape"`
}

type GeoShape struct {
	Location GeoLocationObj `json:"location"`
}

type GeoLocationObj struct {
	Shape    GeoLocation `json:"shape"`
	Relation string      `json:"relation"`
}

// ------------------------------------------------------------------------

type GeoLocationResponse struct {
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total   int       `json:"total"`
	HitList []HitList `json:"hits"`
}

type HitList struct {
	Score  float64      `json:"_score"`
	Source SearchResult `json:"_source"`
}

// SearchResults represents a structure for a list of returned objects
type SearchResults struct {
	Count  int            `json:"count"`
	Items  []SearchResult `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// SearchResult represents data on a single item of search results
type SearchResult struct {
	Name     string      `json:"name"`
	Location GeoLocation `json:"location,omitempty"`
}

// ErrorInvalidRelationValue - return error
func ErrorInvalidRelationValue(m string) error {
	err := errors.New(`incorrect relation value: ` + m + `. It Should be either "within" or "intersects"`)
	return err
}

var validRelation = map[string]bool{
	"intersects": true,
	"within":     true,
}

// ValidateRelation checks the requested relation value is a valid value
func ValidateRelation(relation string) (string, error) {
	r := strings.ToLower(relation)
	if !validRelation[r] {
		return "", ErrorInvalidRelationValue(relation)
	}

	return r, nil
}
