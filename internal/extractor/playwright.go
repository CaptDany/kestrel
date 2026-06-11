package extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type PlaywrightExtractor struct {
	ScraperURL string
	client     *http.Client
}

func NewPlaywrightExtractor(scraperURL string) *PlaywrightExtractor {
	return &PlaywrightExtractor{
		ScraperURL: scraperURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *PlaywrightExtractor) Name() string { return "playwright" }
func (p *PlaywrightExtractor) Supports(u *url.URL) bool {
	return p.ScraperURL != ""
}

type scrapeRequest struct {
	URL string `json:"url"`
}

func (p *PlaywrightExtractor) Extract(rawURL string) (*Result, error) {
	body, _ := json.Marshal(scrapeRequest{URL: rawURL})
	resp, err := p.client.Post(p.ScraperURL+"/api/scrape", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("playwright scrape request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode playwright response: %w", err)
	}
	result.URL = rawURL
	return &result, nil
}
