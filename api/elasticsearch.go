package api

import (
	"context"

	"github.com/ONSdigital/dp-census-search-prototypes/models"
)

// Elasticsearcher - An interface used to access elasticsearch
type Elasticsearcher interface {
	GetPostcodes(ctx context.Context, indexName, postcode string) (*models.PostcodeResponse, int, error)
	QueryGeoLocation(ctx context.Context, indexName string, geoLocation *models.GeoLocation, limit, offset int, relation string) (*models.GeoLocationResponse, int, error)
}
