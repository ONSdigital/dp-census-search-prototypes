package models

type PostcodeDoc struct {
	Postcode    string `json:"postcode"`
	PostcodeRaw string `json:"postcode_raw"`
	Pin         PinObj `json:"pin"`
}

type PinObj struct {
	PinLocation CoordinatePoint `json:"location"`
}

type CoordinatePoint struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type PostcodeRequest struct {
	Query PostcodeQuery `json:"query"`
}

type PostcodeQuery struct {
	Distance PostcodeTerm `json:"term"`
}

type PostcodeTerm struct {
	Postcode string `json:"postcode"`
}

type PostcodeResponse struct {
	Hits []HitObj `json:"hits.hits"`
}

// type HitsObj struct {
// 	Hits []HitObj `json:"hits"`
// }

type HitObj struct {
	Postcode    string  `json:"_source.postcode"`
	RawPostcode string  `json:"_source.postcode_raw"`
	Lat         float64 `json:"_source.pin.location.lat"`
	Lon         float64 `json:"_source.pin.location.lon"`
}
