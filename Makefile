SHELL=bash

BUILD=build
BIN_DIR?=.

PARENT_SEARCH=parents-search
POSTCODE_SEARCH=postcode-search

build:
	go generate ./...
	@mkdir -p $(BUILD)/$(BIN_DIR)

parentsearchbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(PARENT_SEARCH) cmd/$(PARENT_SEARCH)/main.go

postcodesearchbuild: build
	go build -o $(BUILD)/$(BIN_DIR)/$(POSTCODE_SEARCH) cmd/$(POSTCODE_SEARCH)/main.go

parentsearch: parentsearchbuild
	HUMAN_LOG=1 go run -race cmd/$(PARENT_SEARCH)/main.go

postcodesearch: postcodesearchbuild
	HUMAN_LOG=1 go run -race cmd/$(POSTCODE_SEARCH)/main.go

test:
	go test -cover -race ./...

.PHONY: build parentsearch postcodesearch test
