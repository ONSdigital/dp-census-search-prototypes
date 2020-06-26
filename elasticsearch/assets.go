//go:generate go get github.com/jteeuwen/go-bindata/go-bindata
//go:generate go-bindata -pkg elasticsearch ./parent-mappings.json ./postcode-mappings.json ./boundary-file-mappings.json ./geography-mappings.json

package elasticsearch
