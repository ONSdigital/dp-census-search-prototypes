package api

import (
	"context"

	"github.com/ONSdigital/dp-census-search-prototypes/models"
)

// Elasticsearcher - An interface used to access elasticsearch
type Elasticsearcher interface {
	GetPostcodes(ctx context.Context, postcode string) (*models.PostcodeResponse, int, error)
	QueryGeoLocation(ctx context.Context, geodoc string, limit, offset int) (*models.SearchResponse, int, error)
}
