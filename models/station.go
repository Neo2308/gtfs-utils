package models

type Station struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Lat  string `json:"lat"`
	Lng  string `json:"lng"`
}
