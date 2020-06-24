SHELL=bash

BUILD=build
BIN_DIR?=.

PARENT_SEARCH=parents-search
SEARCH_API=search-api
INDEX_CREATION=boundary-file-index

build:
	go generate ./...
	@mkdir -p $(BUILD)/$(BIN_DIR)

parentsearchbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(PARENT_SEARCH) cmd/$(PARENT_SEARCH)/main.go

boundaryindexbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(INDEX_CREATION) cmd/$(INDEX_CREATION)/main.go

apibuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(SEARCH_API) cmd/$(SEARCH_API)/main.go

parentsearch: parentsearchbuild
	HUMAN_LOG=1 go run -race cmd/$(PARENT_SEARCH)/main.go

boundaryindex: boundaryindexbuild
	HUMAN_LOG=1 go run -race cmd/$(INDEX_CREATION)/main.go

api: boundaryindex apibuild
	HUMAN_LOG=1 go run -race cmd/$(SEARCH_API)/main.go

test:
	go test -cover -race ./...

.PHONY: build parentsearch postcodesearch test
