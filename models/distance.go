package models

import (
	"context"
	"errors"
	"strconv"
	"strings"

	errs "github.com/ONSdigital/dp-census-search-prototypes/apierrors"
	"github.com/ONSdigital/log.go/log"
)

// DistObj ...
type DistObj struct {
	Value float64
	Unit  string
}

var kilometres = map[string]bool{
	"km":         true,
	"kilometers": true,
	"kilometer":  true,
	"kilometre":  true,
	"kilometres": true,
}

var miles = map[string]bool{
	"m":     true,
	"miles": true,
}

// ErrorInvalidDistance - return error
func ErrorInvalidDistance(m string) error {
	err := errors.New("invalid distance value: " + m + ". Should contain a number and unit of distance separated by a comma e.g. 40km")
	return err
}

// ValidateDistance ...
func ValidateDistance(distance string) (*DistObj, error) {
	if distance == "" {
		return nil, errs.ErrEmptyDistanceTerm
	}

	lcDistance := strings.ToLower(distance)

	values := strings.SplitAfter(lcDistance, ",")

	if len(values) == 1 {
		return nil, ErrorInvalidDistance(distance)
	}

	if len(values) > 2 {
		return nil, ErrorInvalidDistance(distance)
	}

	value, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return nil, ErrorInvalidDistance(distance)
	}

	if !kilometres[values[1]] && !miles[values[1]] {
		return nil, ErrorInvalidDistance(distance)
	}

	distObj := &DistObj{
		Value: value,
		Unit:  values[1],
	}

	return distObj, nil
}

func (dO *DistObj) CalculateDistanceInMetres(ctx context.Context) (distance float64) {

	switch {
	case kilometres[dO.Unit]:
		distance = dO.Value * 1000
	case miles[dO.Unit]:
		distance = dO.Value * 1609.34
	default:
		log.Event(ctx, "unrecognizable unit value: defaulting to kilometres", log.WARN)
		distance = dO.Value * 1000
	}

	return
}
