package models

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
)

type BoundaryFileRequest struct {
	Query BoundaryFileQuery `json:"query"`
}

type BoundaryFileQuery struct {
	Term BoundaryFileTerm `json:"term"`
}

type BoundaryFileTerm struct {
	ID string `json:"id"`
}

// ------------------------------------------------------------------------

type BoundaryFileResponse struct {
	Hits EmbededHits `json:"hits"`
}

// ------------------------------------------------------------------------

type BoundaryDoc struct {
	ID       string      `json:"id"`
	Location GeoLocation `json:"location"`
}

// ------------------------------------------------------------------------

var validTypes = map[string]bool{
	"polygon":      true,
	"multipolygon": true,
}

// CreateGeoLocation manages the creation of a geo location from a reader
func CreateGeoLocation(reader io.Reader) (*GeoLocation, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errs.ErrUnableToReadMessage
	}

	var geoLocation GeoLocation

	err = json.Unmarshal(b, &geoLocation)
	if err != nil {
		return nil, errs.ErrUnableToParseJSON
	}
	return &geoLocation, nil
}

// ErrorInvalidType - return error
func ErrorInvalidType(m string) error {
	err := errors.New("invalid type value: " + m + ". Should be one of the following: polygon")
	return err
}

// ValidateShape ...
func ValidateShape(geoLocation *GeoLocation) error {
	if err := ValidateType(geoLocation.Type); err != nil {
		return err
	}

	if err := ValidateShapeFile(geoLocation.Type, geoLocation.Coordinates); err != nil {
		return err
	}

	return nil
}

// ValidateType ...
func ValidateType(shapeType string) error {
	if shapeType == "" {
		return errs.ErrMissingType
	}

	if !validTypes[shapeType] {
		return ErrorInvalidType(shapeType)
	}

	return nil
}

// ValidateShapeFile ...
func ValidateShapeFile(gType string, shapeFile interface{}) error {
	if shapeFile == nil {
		return errs.ErrMissingShapeFile
	}

	if gType == "multipolygon" {

		// geometry := shapeFile.([][][][]float64)
	}

	if gType == "polygon" {
		geometry := shapeFile.([][][]float64)

		for _, shape := range geometry {
			if shape == nil || len(shape) < 1 {
				return errs.ErrEmptyShape
			}

			if len(shape) < 4 {
				return errs.ErrLessThanFourCoordinates
			}

			lastIndex := 0
			for i, coordinates := range shape {
				if coordinates == nil {
					return errs.ErrEmptyCoordinates
				}

				// Check coordinates have exactly two values, lat/long
				if len(coordinates) != 2 {
					return errs.ErrInvalidCoordinates
				}

				lastIndex = i
			}

			// Check first and last coordinate are the same
			if shape[0][0] != shape[lastIndex][0] || shape[0][1] != shape[lastIndex][1] {
				return errs.ErrInvalidShape
			}
		}
	}

	return nil
}
