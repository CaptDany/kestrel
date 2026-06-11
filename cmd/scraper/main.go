package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	playwright "github.com/playwright-community/playwright-go"
)

type scrapeRequest struct {
	URL string `json:"url"`
}

type scrapeResult struct {
	Title    string   `json:"title"`
	Price    *float64 `json:"price"`
	Currency string   `json:"currency"`
	URL      string   `json:"url"`
}

var pw *playwright.Playwright
var browser playwright.Browser

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	var err error
	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("start playwright: %v", err)
	}

	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("launch browser: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/scrape", handleScrape)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("scraper starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, 405, "method not allowed")
		return
	}

	var req scrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonError(w, 400, "url required")
		return
	}

	page, err := browser.NewPage()
	if err != nil {
		jsonError(w, 500, "create page: "+err.Error())
		return
	}
	defer func() { _ = page.Close() }()

	if _, err := page.Goto(req.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(15000),
	}); err != nil {
		jsonError(w, 422, "navigate: "+err.Error())
		return
	}

	content, err := page.Content()
	if err != nil {
		jsonError(w, 500, "get content: "+err.Error())
		return
	}

	result := extractFromHTML(content, req.URL)
	jsonResp(w, 200, result)
}

func extractFromHTML(html, rawURL string) *scrapeResult {
	res := &scrapeResult{URL: rawURL}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return res
	}

	res.Title = strings.TrimSpace(doc.Find("title").Text())
	if t := strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", "")); t != "" {
		res.Title = t
	}

	priceStr := doc.Find(`meta[property="product:price:amount"]`).AttrOr("content", "")
	if priceStr == "" {
		priceStr = doc.Find(`meta[property="og:price:amount"]`).AttrOr("content", "")
	}

	if priceStr != "" {
		var p float64
		if _, err := fmt.Sscanf(priceStr, "%f", &p); err == nil {
			res.Price = &p
		}
	}

	res.Currency = doc.Find(`meta[property="product:price:currency"]`).AttrOr("content", "")
	if res.Currency == "" {
		res.Currency = doc.Find(`meta[property="og:price:currency"]`).AttrOr("content", "")
	}
	if res.Currency == "" {
		res.Currency = "USD"
	}

	return res
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, map[string]string{"status": "ok"})
}

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResp(w, status, map[string]string{"error": msg})
}
