SHELL=bash

BUILD=build
BIN_DIR?=.

PARENT_SEARCH=parents-search
POSTCODE_SEARCH=postcode-search
SEARCH_API=search-api
INDEX_CREATION=boundary-file-index
GEO_BOUNDARIES=geo-boundaries

build:
	go generate ./...
	@mkdir -p $(BUILD)/$(BIN_DIR)

parentsearchbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(PARENT_SEARCH) cmd/$(PARENT_SEARCH)/main.go

postcodesearchbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(POSTCODE_SEARCH) cmd/$(POSTCODE_SEARCH)/main.go

boundaryindexbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(INDEX_CREATION) cmd/$(INDEX_CREATION)/main.go

apibuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(SEARCH_API) cmd/$(SEARCH_API)/main.go

geoboundariesbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(GEO_BOUNDARIES) cmd/$(GEO_BOUNDARIES)/main.go

parentsearch: parentsearchbuild
	HUMAN_LOG=1 go run -race cmd/$(PARENT_SEARCH)/main.go

postcodesearch: postcodesearchbuild
	HUMAN_LOG=1 go run -race cmd/$(POSTCODE_SEARCH)/main.go

boundaryindex: boundaryindexbuild
	HUMAN_LOG=1 go run -race cmd/$(INDEX_CREATION)/main.go

geoboundaries: geoboundariesbuild
	HUMAN_LOG=1 go run -race cmd/$(GEO_BOUNDARIES)/main.go

api: boundaryindex apibuild
	HUMAN_LOG=1 go run -race cmd/$(SEARCH_API)/main.go

test:
	go test -cover -race ./...

.PHONY: build parentsearch postcodesearch test
