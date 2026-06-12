package extractor

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type amazonParser struct{}

func (a *amazonParser) Name() string { return "amazon" }
func (a *amazonParser) Supports(u *url.URL) bool {
	return strings.Contains(u.Hostname(), "amazon.")
}

func (a *amazonParser) Extract(rawURL string) (*Result, error) {
	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}
	res := &Result{URL: rawURL}

	res.Title = strings.TrimSpace(doc.Find("#productTitle").Text())
	res.ImageURL, _ = doc.Find("#landingImage").Attr("src")
	if res.ImageURL == "" {
		res.ImageURL, _ = doc.Find("#imgTagWrapperId img").Attr("src")
	}
	if res.ImageURL == "" {
		res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")
	}

	whole := strings.TrimSpace(doc.Find(".a-price-whole").First().Text())
	fraction := strings.TrimSpace(doc.Find(".a-price-fraction").First().Text())
	if whole != "" {
		priceStr := whole
		if fraction != "" {
			priceStr += "." + fraction
		}
		priceStr = strings.ReplaceAll(priceStr, ",", "")
		if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
			res.Price = &p
		}
	}

	if res.Price == nil {
		res.Price, res.Currency = extractJSONLD(doc)
	}

	if res.Price == nil {
		extractMeta(doc, res)
	}

	if res.Price == nil {
		sym := strings.TrimSpace(doc.Find(".a-price-symbol").First().Text())
		if sym != "" {
			priceStr := strings.TrimSpace(doc.Find(".a-offscreen").First().Text())
			priceStr = strings.TrimPrefix(priceStr, sym)
			priceStr = strings.ReplaceAll(priceStr, ",", "")
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				res.Price = &p
			}
		}
	}

	res.Currency = detectCurrency(rawURL)
	return res, nil
}

type mercadolibreParser struct{}

func (m *mercadolibreParser) Name() string { return "mercadolibre" }
func (m *mercadolibreParser) Supports(u *url.URL) bool {
	return strings.Contains(u.Hostname(), "mercadolibre.") || strings.Contains(u.Hostname(), "mercadolivre.")
}

func (m *mercadolibreParser) Extract(rawURL string) (*Result, error) {
	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}
	res := &Result{URL: rawURL}

	res.Title = strings.TrimSpace(doc.Find("h1.ui-pdp-title").Text())
	if res.Title == "" {
		res.Title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	}

	res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	p, c := extractJSONLD(doc)
	if p != nil {
		res.Price = p
		res.Currency = c
	}

	if res.Price == nil {
		metaPrice := strings.TrimSpace(doc.Find(`meta[itemprop="price"]`).AttrOr("content", ""))
		if metaPrice != "" {
			if parsed, err := strconv.ParseFloat(metaPrice, 64); err == nil {
				res.Price = &parsed
			}
		}
	}

	if res.Currency == "" {
		res.Currency = strings.TrimSpace(doc.Find(`meta[itemprop="priceCurrency"]`).AttrOr("content", ""))
	}

	if res.Price == nil {
		extractMeta(doc, res)
	}

	if res.Currency == "" {
		res.Currency = detectCurrency(rawURL)
	}
	return res, nil
}

type aliexpressParser struct{}

func (a *aliexpressParser) Name() string { return "aliexpress" }
func (a *aliexpressParser) Supports(u *url.URL) bool {
	return strings.Contains(u.Hostname(), "aliexpress.") || strings.Contains(u.Hostname(), "aliexpress.us")
}

var aliexpressPriceRe = regexp.MustCompile(`"skuVal".*?"actPrice"[^"]*"[^"]*?"(\d+\.?\d*)"`)

func (a *aliexpressParser) Extract(rawURL string) (*Result, error) {
	res := &Result{URL: rawURL}

	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}

	res.Title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	if res.Title == "" {
		res.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	html, _ := doc.Html()
	if match := aliexpressPriceRe.FindStringSubmatch(html); len(match) > 1 {
		if p, err := strconv.ParseFloat(match[1], 64); err == nil {
			res.Price = &p
		}
	}

	if res.Price == nil {
		res.Price, res.Currency = extractJSONLD(doc)
	}
	if res.Price == nil {
		extractMeta(doc, res)
	}

	res.Currency = detectCurrency(rawURL)
	return res, nil
}

type ikeaParser struct{}

func (i *ikeaParser) Name() string { return "ikea" }
func (i *ikeaParser) Supports(u *url.URL) bool {
	return strings.Contains(u.Hostname(), "ikea.")
}

func (i *ikeaParser) Extract(rawURL string) (*Result, error) {
	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}
	res := &Result{URL: rawURL}

	res.Title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	if res.Title == "" {
		res.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	p, c := extractJSONLD(doc)
	if p != nil {
		res.Price = p
		res.Currency = c
	}

	if res.Price == nil {
		extractMeta(doc, res)
	}

	if res.Currency == "" {
		res.Currency = detectCurrency(rawURL)
	}
	return res, nil
}

type walmartParser struct{}

func (w *walmartParser) Name() string { return "walmart" }
func (w *walmartParser) Supports(u *url.URL) bool {
	return strings.Contains(u.Hostname(), "walmart.")
}

func (w *walmartParser) Extract(rawURL string) (*Result, error) {
	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}
	res := &Result{URL: rawURL}

	res.Title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	if res.Title == "" {
		res.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	p, c := extractJSONLD(doc)
	if p != nil {
		res.Price = p
		res.Currency = c
	}

	if res.Price == nil {
		priceStr := strings.TrimSpace(doc.Find(`[itemprop="price"]`).AttrOr("content", ""))
		if priceStr == "" {
			priceStr = strings.TrimSpace(doc.Find(".price-1LNHH").First().Text())
		}
		if priceStr != "" {
			priceStr = strings.TrimPrefix(priceStr, "$")
			priceStr = strings.ReplaceAll(priceStr, ",", "")
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				res.Price = &p
			}
		}
	}

	if res.Price == nil {
		extractMeta(doc, res)
	}

	if res.Currency == "" {
		res.Currency = detectCurrency(rawURL)
	}
	return res, nil
}

type genericParser struct{}

func (g *genericParser) Name() string { return "generic" }
func (g *genericParser) Supports(u *url.URL) bool {
	return true
}

func (g *genericParser) Extract(rawURL string) (*Result, error) {
	doc, err := fetchDoc(rawURL)
	if err != nil {
		return nil, err
	}
	res := &Result{URL: rawURL}

	res.Title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	if res.Title == "" {
		res.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	res.ImageURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	p, c := extractJSONLD(doc)
	if p != nil {
		res.Price = p
		res.Currency = c
	}

	if res.Price == nil {
		extractMeta(doc, res)
	}

	if res.Price == nil {
		priceRegex := regexp.MustCompile(`[$€£¥]\s*(\d{1,3}(?:,\d{3})*(?:\.\d{1,2})?)`)
		body := doc.Find("body").Text()
		if match := priceRegex.FindStringSubmatch(body); len(match) > 1 {
			priceStr := strings.ReplaceAll(match[1], ",", "")
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				res.Price = &p
			}
		}
	}

	if res.Currency == "" {
		res.Currency = detectCurrency(rawURL)
	}
	return res, nil
}

func GetParser(u *url.URL) Extractor {
	parsers := []Extractor{
		&amazonParser{},
		&mercadolibreParser{},
		&aliexpressParser{},
		&ikeaParser{},
		&walmartParser{},
	}
	for _, p := range parsers {
		if p.Supports(u) {
			return p
		}
	}
	return &genericParser{}
}

func GetParserByName(name string) Extractor {
	parsers := []Extractor{
		&amazonParser{},
		&mercadolibreParser{},
		&aliexpressParser{},
		&ikeaParser{},
		&walmartParser{},
		&genericParser{},
	}
	for _, p := range parsers {
		if p.Name() == name {
			return p
		}
	}
	return &genericParser{}
}

func detectCurrency(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "USD"
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(host, "mercadolibre.com.mx"),
		strings.Contains(host, "amazon.com.mx"):
		return "MXN"
	case strings.Contains(host, "mercadolibre.com.co"):
		return "COP"
	case strings.Contains(host, "mercadolibre.com.ar"),
		strings.Contains(host, "amazon.com.ar"):
		return "ARS"
	case strings.Contains(host, "mercadolibre.cl"):
		return "CLP"
	case strings.Contains(host, "mercadolibre.com.pe"):
		return "PEN"
	case strings.Contains(host, "mercadolibre.com.br"),
		strings.Contains(host, "mercadolivre.com.br"):
		return "BRL"
	case strings.Contains(host, "amazon.co.uk"),
		strings.Contains(host, "ikea.com/gb"):
		return "GBP"
	case strings.Contains(host, "amazon.de"),
		strings.Contains(host, "amazon.fr"),
		strings.Contains(host, "amazon.it"),
		strings.Contains(host, "amazon.es"),
		strings.Contains(host, "ikea.com/de"),
		strings.Contains(host, "ikea.com/fr"),
		strings.Contains(host, "ikea.com/it"),
		strings.Contains(host, "ikea.com/es"):
		return "EUR"
	case strings.Contains(host, "amazon.co.jp"):
		return "JPY"
	case strings.Contains(host, "amazon.ca"),
		strings.Contains(host, "walmart.ca"):
		return "CAD"
	case strings.Contains(host, "amazon.com.au"):
		return "AUD"
	default:
		return "USD"
	}
}

func extractJSONLD(doc *goquery.Document) (*float64, string) {
	var price *float64
	var currency string

	doc.Find("script[type='application/ld+json']").Each(func(_ int, s *goquery.Selection) {
		if price != nil {
			return
		}
		text := s.Text()
		re := regexp.MustCompile(`"price"[^:]*:\s*"(\d+\.?\d*)"`)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			if p, err := strconv.ParseFloat(m[1], 64); err == nil {
				price = &p
			}
		}
		if m := re.FindStringSubmatch(text); len(m) > 1 && price == nil {
			if p, err := strconv.ParseFloat(m[1], 64); err == nil {
				price = &p
			}
		}
		reCurr := regexp.MustCompile(`"priceCurrency"[^:]*:\s*"([A-Z]{3})"`)
		if m := reCurr.FindStringSubmatch(text); len(m) > 1 {
			currency = m[1]
		}
	})
	return price, currency
}

func extractMeta(doc *goquery.Document, res *Result) {
	metaPrice := doc.Find(`meta[property="product:price:amount"]`).AttrOr("content", "")
	if metaPrice == "" {
		metaPrice = doc.Find(`meta[property="og:price:amount"]`).AttrOr("content", "")
	}
	if metaPrice == "" {
		metaPrice = doc.Find(`meta[name="price"]`).AttrOr("content", "")
	}
	if metaPrice != "" {
		metaPrice = strings.ReplaceAll(metaPrice, ",", "")
		if p, err := strconv.ParseFloat(metaPrice, 64); err == nil {
			res.Price = &p
		}
	}

	metaCurr := doc.Find(`meta[property="product:price:currency"]`).AttrOr("content", "")
	if metaCurr == "" {
		metaCurr = doc.Find(`meta[property="og:price:currency"]`).AttrOr("content", "")
	}
	if metaCurr == "" {
		metaCurr = doc.Find(`meta[name="currency"]`).AttrOr("content", "")
	}
	if metaCurr != "" {
		res.Currency = metaCurr
	}
}

func fetchDoc(rawURL string) (*goquery.Document, error) {
	reader, err := fetchURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return doc, nil
}
