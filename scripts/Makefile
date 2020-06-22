SHELL=bash

BUILD=build
BIN_DIR?=.

REFRESH=refresh
LSOA=2011-lsoa
MSOA=2011-msoa
OA=2011-oa
TCITY=2015-tcity
GEOJSON=geojson

build:
	go generate ../...
	@mkdir -p ../$(BUILD)/$(BIN_DIR)

refreshbuild: build
	go build -o ../$(BUILD)/$(BIN_DIR)/$(REFRESH) $(GEOJSON)/$(REFRESH)/main.go

refresh: refreshbuild
	HUMAN_LOG=1 go run -race $(GEOJSON)/$(REFRESH)/main.go

lsoabuild: build
	go build -o ../$(BUILD)/$(BIN_DIR)/$(LSOA) $(GEOJSON)/$(LSOA)/main.go

msoabuild: build
	go build -o ../$(BUILD)/$(BIN_DIR)/$(MSOA) $(GEOJSON)/$(MSOA)/main.go

oabuild: build
	go build -o ../$(BUILD)/$(BIN_DIR)/$(OA) $(GEOJSON)/$(OA)/main.go

tcitybuild: build
	go build -o ../$(BUILD)/$(BIN_DIR)/$(TCITY) $(GEOJSON)/$(TCITY)/main.go

lsoa: lsoabuild
	HUMAN_LOG=1 go run -race $(GEOJSON)/$(LSOA)/main.go

msoa: msoabuild
	HUMAN_LOG=1 go run -race $(GEOJSON)/$(MSOA)/main.go

oa: oabuild
	HUMAN_LOG=1 go run -race $(GEOJSON)/$(OA)/main.go

tcity: tcitybuild
	HUMAN_LOG=1 go run -race $(GEOJSON)/$(TCITY)/main.go

geojson: refresh lsoa msoa tcity oa

test:
	go test -cover -race ./...

.PHONY: build geojson lsoa msoa tcity refresh test
