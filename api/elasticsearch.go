package api

import (
	"context"

	"github.com/ONSdigital/dp-census-search-prototypes/models"
)

// Elasticsearcher - An interface used to access elasticsearch
type Elasticsearcher interface {
	AddBoundaryFile(ctx context.Context, indexName string, boundaryDoc *models.BoundaryDoc) (int, error)
	GetBoundaryFile(ctx context.Context, indexName, id string) (*models.BoundaryFileResponse, int, error)
	GetBoundaryFiles(ctx context.Context, indexName string, query interface{}) (*models.GeoResponseWithLocation, int, error)
	GetPostcodes(ctx context.Context, indexName, postcode string) (*models.PostcodeResponse, int, error)
	QueryGeoLocation(ctx context.Context, indexName string, geoLocation *models.GeoLocation, limit, offset int, relation string) (*models.GeoResponse, int, error)
}
