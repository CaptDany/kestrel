package extractor

import "net/url"

type Result struct {
	Title    string   `json:"title"`
	Price    *float64 `json:"price"`
	Currency string   `json:"currency"`
	ImageURL string   `json:"image_url"`
	URL      string   `json:"url"`
}

type Extractor interface {
	Name() string
	Supports(u *url.URL) bool
	Extract(rawURL string) (*Result, error)
}
