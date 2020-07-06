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
		geometry := shapeFile.([]interface{})

		for _, p := range geometry {
			if p == nil {
				return errs.ErrEmptyShape
			}
			polygons := p.([]interface{})

			if len(polygons) < 2 {
				return errs.ErrLessThanTwoPolygons
			}

			for _, s := range polygons {
				if s == nil {
					return errs.ErrEmptyShape
				}
				shapes := s.([]interface{})

				if len(shapes) < 4 {
					return errs.ErrLessThanFourCoordinates
				}

				var firstShape, lastShape []float64
				for i, c := range shapes {
					lastShape = nil
					if c == nil {
						return errs.ErrEmptyCoordinates
					}
					coordinates := c.([]interface{})

					if len(coordinates) != 2 {
						return errs.ErrInvalidCoordinates
					}

					for _, coordinate := range coordinates {
						lastShape = append(lastShape, coordinate.(float64))
					}

					if i == 0 {
						firstShape = lastShape
					}
				}

				// Check first and last coordinate are the same
				if firstShape[0] != lastShape[0] || firstShape[1] != lastShape[1] {
					return errs.ErrInvalidShape
				}
			}
		}
	}

	if gType == "polygon" {
		geometry := shapeFile.([]interface{})

		for _, s := range geometry {
			if s == nil {
				return errs.ErrEmptyShape
			}
			shapes := s.([]interface{})

			if len(shapes) < 4 {
				return errs.ErrLessThanFourCoordinates
			}

			var firstShape, lastShape []float64
			for i, c := range shapes {
				lastShape = nil
				if c == nil {
					return errs.ErrEmptyCoordinates
				}
				coordinates := c.([]interface{})

				if len(coordinates) != 2 {
					return errs.ErrInvalidCoordinates
				}

				for _, coordinate := range coordinates {
					lastShape = append(lastShape, coordinate.(float64))
				}

				if i == 0 {
					firstShape = lastShape
				}
			}

			// Check first and last coordinate are the same
			if firstShape[0] != lastShape[0] || firstShape[1] != lastShape[1] {
				return errs.ErrInvalidShape
			}
		}
	}

	return nil
}
