package models

type GeoDoc struct {
	Name     string      `json:"name"`
	Location GeoLocation `json:"location"`
}

type GeoLocation struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}
