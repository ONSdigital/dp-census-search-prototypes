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
	Count      int            `json:"count"`
	Items      []SearchResult `json:"items"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	TotalCount int            `json:"total_count"`
}

// SearchResult represents data on a single item of search results
type SearchResult struct {
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Hierarchy    string  `json:"hierarchy"`
	LSOA11NM     string  `json:"lsoa11nm,omitempty"`
	LSOA11NMW    string  `json:"lsoa11nmw,omitempty"`
	MSOA11NM     string  `json:"msoa11nm,omitempty"`
	MSOA11NMW    string  `json:"msoa11nmw,omitempty"`
	ShapeArea    float64 `json:"shape_area,omitempty"`
	ShapeLength  float64 `json:"shape_length,omitempty"`
	StatedArea   float64 `json:"stated_area,omitempty"`
	StatedLength float64 `json:"stated_length,omitempty"`
	TCITY15NM    string  `json:"tcity15nm,omitempty"`
	// Location     OutputGeoLocation `json:"location,omitempty"`
}

type OutputGeoLocation struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
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
