package api

import (
	"context"

	"github.com/ONSdigital/dp-census-search-prototypes/models"
)

// Elasticsearcher - An interface used to access elasticsearch
type Elasticsearcher interface {
	GetBoundaryFile(ctx context.Context, indexName, id string) (*models.BoundaryFileResponse, int, error)
	GetPostcodes(ctx context.Context, indexName, postcode string) (*models.PostcodeResponse, int, error)
	QueryGeoLocation(ctx context.Context, indexName string, geoLocation *models.GeoLocation, limit, offset int, relation string) (*models.GeoLocationResponse, int, error)
}
