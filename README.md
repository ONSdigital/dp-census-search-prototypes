# dp-census-search-prototypes
Several search prototypes for census 2021

## Requirements

In order to run the service locally you will need the following:

- Go
- Git
- ElasticSearch (version 6.7 or 6.8)

### Getting started

- Clone the repo go get github.com/ONSdigital/dp-census-search-prototypes
- Run elasticsearch
- Choose an application/script to run
    - Generate searchable documents across geographical boundaries and find parent resources (e.g. find Wales resources if searching for Cardiff), see [documentation here](#geographical-search-including-parent-documents)
    - Generate postcode index to search for datasets by postcode, see [documentation here](#postcode-search-with-distance)
    - Generate search API for accessing prototypes, see [documentation here](#search-api)

#### Notes

See [command list](COMMANDS.md) for a list of helpful commands to run alongside and independently from scripts or prototypes.

### Geographical Search (including parent documents)

#### Setting up data

Using `scripts/test-data/datasets.csv` file to upload geo location docs into Elasticsearch, data in here is made up but structurally based on what data we do have in the geoportal (example [here](https://geoportal.statistics.gov.uk/datasets/london-assembly-constituencies-december-2018-boundaries-en-bfc/geoservice)) and what is expected by Elasticsearch.

The model works for versions 6.7 and 6.8. A slight tweak to the mappings.json file to get it working with version 7.*.* by removing extra nest of `doc`.

7 documents will be generated and stored on an elasticsearch index of `test_parent` by running `cd scripts; make parent; cd ..`

#### GeoLocation Queries

1) Within Cardiff boundaries (from test file)

```
curl -X GET "localhost:9200/test_parent/_search?pretty" -H 'Content-Type: application/json' -d'
{
    "query":{
        "bool": {
            "must": {
                "match_all": {}
            },
            "filter": {
                "geo_shape": {
                    "location": {
                        "shape": {
                            "type": "polygon",
                            "coordinates" : [[[-3.232257,51.507306], [-3.128257,51.500306], [-3.136840,51.467705], [-3.2085046,51.4520104], [-3.232257,51.507306]]]
                        },
                        "relation": "within"
                    }
                }
            }
        }
    }
}
'
```

Areas in which the boundaries intersect the Cardiff boundaries will not be returned, here we will see only Cathays and Roath documents returned as Canton boundary file in test.csv crosses over Cardiff boundary. Wales wont be returned because it is not *within* the Cardiff boundary.

2) Within Cathays boundaries (from test file)

```
curl -X GET "localhost:9200/test_parent/_search?pretty" -H 'Content-Type: application/json' -d'
{
    "query":{
        "bool": {
            "must": {
                "match_all": {}
            },
            "filter": {
                "geo_shape": {
                    "location": {
                        "shape": {
                            "type": "polygon",
                            "coordinates" : [[[-3.18280,51.4963], [-3.1780,51.5003], [-3.1640,51.4943], [-3.1750,51.4883], [-3.18280,51.4963]]]
                        },
                        "relation": "intersects"
                    }
                }
            }
        }
    }
}
'
```

3) Demonstration using a multipolygon search:


```
curl -X GET "localhost:9200/test_parent/_search?pretty" -H 'Content-Type: application/json' -d'
{
    "query":{
        "bool": {
            "must": {
                "match_all": {}
            },
            "filter": {
                "geo_shape": {
                    "location": {
                        "shape": {
                            "type": "multipolygon",
                            "coordinates" : [[[[-3.18280,51.4963], [-3.1780,51.5003], [-3.1640,51.4943], [-3.1750,51.4883], [-3.18280,51.4963]]]]
                        },
                        "relation": "intersects"
                    }
                }
            }
        }
    }
}
'
```

Intersects will find all boundaries which cross over with the polygon above - scoring is equal so cannot distinguish smaller areas which should be higher in the list then larger areas, e.g. Cathays, Roath, Cardiff then Wales.

### Postcode Search with Distance

Search for datasets within a distance of postcode.

#### Setting up data

1) Download postcode data from the geo portal [here](https://geoportal.statistics.gov.uk/datasets/national-statistics-postcode-lookup-february-2020). Then click on download and move zip to root of this repository on your local machine: `mv Downloads/NSPL_FEB_2020_UK.zip .`. Unzip file, the data layout to postcode data should look like:
    - NSPL_FEB_2020_UK
      - Data
        - NSPL_FEB_2020_UK.csv

2) Upload postcode data to elasticsearch index with:
`cd scripts; make postcode; cd ..`
This will take approximately 4 minutes and 20 seconds and documents will be stored in `test_postcode` index.

#### Postcode Queries

1) Find postcode and return latitude, longitude coordinate to be use to find datasets.

```
curl -XGET localhost:9200/test_postcode/_search  -H 'Content-Type: application/json' -d'
{
    "query": {
        "term": {
            "postcode": "ze39xp"
        }
    }
}
'
```
2) Generate polygon shape boundary based on circle defined by postcode coordinate and distance.

3) Find datasets that are within the generated polygon circle:

```
curl -X GET "localhost:9200/test_parent/_search?pretty" -H 'Content-Type: application/json' -d'
{
    "query":{
        "bool": {
            "must": {
                "match_all": {}
            },
            "filter": {
                "geo_shape": {
                    "location": {
                        "shape": {
                            "type": "polygon",
                            "coordinates" : [<generated polygon>]
                        },
                        "relation": "within"
                    }
                }
            }
        }
    }
}
'
```

### Search API

All prototypes developed will exist on an endpoint in the search API. These include:

- Search by Postcodes - endpoint: GET `/search/postcodes/{postcode}`
- Search for parent docs via geo boundary file:
    - POST `search/parent` with shape file
    - GET `search/parent/{shape_id}`
- Search by Placename - endpoint: GET `/search/placenames/{name}`

See [swagger spec](swagger.yaml) for documentation of how to use each endpoint on the API. Copy yaml into [swagger editor](https://editor.swagger.io/) (left panel) to generate a pretty web ui on the right to navigate documentaion.

#### Setting up data

Follow setting up data for [Geographical Search including parent documents: Setup](#geographical-search-including-parent-documents) and [Postocde Search with Distance: Setup](#geographical-search-including-parent-documents).

To create a larger selection of datasets to possibly return from either GET endpoints. Use the script geojson that will load in over 700,000 geographical areas across England and Wales as described by the census 2011 boundaries. Follow the steps [here](scripts/README.md#load-data-from-geojson-files). This will load data into the `test_geo` index.

Depending on which script you run the data gets added to different indexes `test_geo` or `test_parent`; before running the API you will need to update the configuration of the application to use either of these values for the `DATASET_INDEX` environment variable, this value is defaulted to `test_parent`.

Use either of the following:
- `export DATASET_INDEX=test_geo`
- `export DATASET_INDEX=test_parent` or use `unset DATASET_INDEX` and will fall back to default value

#### Run API

To start up the API use the following command: ...

`make debug`

...in root of repository.

Follow swagger documentation on how to interact with local api, some examples are below:

```
curl -XGET localhost:10000/search/postcodes/BR33DA?distance=5,miles
curl -XGET localhost:10000/search/postcodes/cf244ny?distance=0.5,km&relation=intersects

curl -XPOST localhost:10000/search/parent -d'{
  "type": "polygon",
  "coordinates": [[[-3.232257,51.507306],[-3.128257,51.500306],[-3.13684,51.467704],[-3.2085046,51.4520104],[-3.232257,51.507306]]]
}'
curl -XGET localhost:10000/search/parent/{shape_id}

curl -XGET localhost:10000/search/placenames/bradford
curl -XGET localhost:10000/search/placenames/bradford?limit=1&offset=1
```
