SHELL=bash

BUILD=build
BIN_DIR?=.

PARENT_SEARCH=parents-search

build:
	go generate ./...
	@mkdir -p $(BUILD)/$(BIN_DIR)
	go build -o $(BUILD)/$(BIN_DIR)/$(PARENT_SEARCH) cmd/$(PARENT_SEARCH)/main.go

parentsearch: build
	HUMAN_LOG=1 go run -race cmd/$(PARENT_SEARCH)/main.go

.PHONY: build parentsearch