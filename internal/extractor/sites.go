package extractor

import (
	"encoding/json"
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
		whole := strings.TrimSpace(doc.Find(".andes-money-amount__fraction").First().Text())
		if whole != "" {
			cents := strings.TrimSpace(doc.Find(".andes-money-amount__cents").First().Text())
			priceStr := strings.ReplaceAll(whole, ".", "")
			priceStr = strings.ReplaceAll(priceStr, ",", ".")
			if cents != "" {
				priceStr += "." + cents
			}
			if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
				res.Price = &p
			}
			if res.Currency == "" {
				sym := strings.TrimSpace(doc.Find(".andes-money-amount__currency-symbol").First().Text())
				if sym == "$" {
					sym = ""
				}
				if sym != "" {
					res.Currency = sym
				}
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

// SupportsWishlist returns true if the URL points to a known wishlist page.
func SupportsWishlist(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	if !strings.Contains(host, "amazon.") {
		return false
	}
	path := strings.ToLower(u.Path)
	return strings.Contains(path, "/wishlist/") || strings.Contains(path, "/registry/")
}

// ExtractWishlist scrapes all pages of an Amazon wishlist and returns all items.
func ExtractWishlist(rawURL string) ([]*Result, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if !SupportsWishlist(u) {
		return nil, fmt.Errorf("unsupported wishlist url")
	}

	var allResults []*Result
	page := 1
	seen := make(map[string]bool)

	for {
		pageURL := rawURL
		if page > 1 {
			q := u.Query()
			q.Set("page", strconv.Itoa(page))
			pageURL = u.Scheme + "://" + u.Host + u.Path + "?" + q.Encode()
		}

		doc, err := fetchDoc(pageURL)
		if err != nil {
			return allResults, fmt.Errorf("fetch page %d: %w", page, err)
		}

		var items *goquery.Selection
		if sel := doc.Find("#g-items [data-id]"); sel.Length() > 0 {
			items = sel
		} else if sel := doc.Find("[data-id]"); sel.Length() > 0 {
			items = sel
		} else {
			items = doc.Find(".a-list-item")
		}
		if items.Length() == 0 {
			if page == 1 {
				gItems := doc.Find("#g-items")
				if gItems.Length() == 0 {
					return nil, fmt.Errorf("this wishlist is private or restricted — make it public before importing")
				}
			}
			break
		}

		var pageResults []*Result
		items.Each(func(_ int, s *goquery.Selection) {
			r := &Result{}

				link := s.Find("a[id^='itemName_'], a[class*='product-link'], h2 a, a[class*='item-name'], a[href*='/dp/']").First()
			if link.Length() == 0 {
				s.Find("a").Each(func(_ int, a *goquery.Selection) {
					if link.Length() > 0 {
						return
					}
					t := strings.TrimSpace(a.Text())
					if t != "" && !strings.EqualFold(t, "Vista rápida") && !strings.EqualFold(t, "Quick view") {
						link = a
					}
				})
			}
			if link.Length() == 0 {
				link = s.Find("a").First()
			}

			linkText := strings.TrimSpace(link.Text())
			if linkText != "" && !strings.EqualFold(linkText, "Vista rápida") && !strings.EqualFold(linkText, "Quick view") {
				r.Title = linkText
			}
			if r.Title == "" {
				if label, exists := link.Attr("aria-label"); exists {
					r.Title = strings.TrimSpace(label)
				}
			}
			if r.Title == "" {
				if title, exists := link.Attr("title"); exists {
					r.Title = strings.TrimSpace(title)
				}
			}
			if r.Title == "" {
				img := s.Find("img[src*='images'], img[data-a-dynamic-image]").First()
				if alt, exists := img.Attr("alt"); exists {
					r.Title = strings.TrimSpace(alt)
				}
			}

			var href string
			var hasHref bool
			if href, hasHref = link.Attr("href"); !hasHref || href == "" {
				if l2 := s.Find("a[href*='/dp/']").First(); l2.Length() > 0 {
					link = l2
					href, hasHref = link.Attr("href")
				}
			}
			if hasHref && href != "" {
				if strings.HasPrefix(href, "http") {
					r.URL = href
				} else {
					r.URL = u.Scheme + "://" + u.Host + href
				}
			}

			img := s.Find("img[src*='images'], img[data-a-dynamic-image]").First()
			if src, ok := img.Attr("src"); ok && src != "" {
				r.ImageURL = src
			} else if dyn, ok := img.Attr("data-a-dynamic-image"); ok && dyn != "" {
				r.ImageURL = firstImageURL(dyn)
			}

			priceRe := regexp.MustCompile(`[\d,]+\.?\d*`)
			priceText := strings.TrimSpace(s.Find(".a-price .a-offscreen").First().Text())
			if priceText == "" {
				priceText = strings.TrimSpace(s.Find(".a-price-whole").First().Text())
				frac := strings.TrimSpace(s.Find(".a-price-fraction").First().Text())
				if frac != "" {
					priceText += "." + frac
				}
			}
			if priceText == "" {
				priceText = strings.TrimSpace(s.Find(".a-price").First().Text())
			}
			if priceText == "" {
				priceText = strings.TrimSpace(s.Find(".a-color-price").First().Text())
			}
			if priceText == "" {
				if el := s.Find("[data-a-price]"); el.Length() > 0 {
					if v, exists := el.Attr("data-a-price"); exists {
						priceText = v
					}
				}
			}
			if priceText == "" {
				if el := s.Find("[data-grid-add-to-cart]"); el.Length() > 0 {
					if v, exists := el.Attr("data-grid-add-to-cart"); exists {
						var data struct {
							Price string `json:"price"`
						}
						if err := json.Unmarshal([]byte(v), &data); err == nil && data.Price != "" {
							priceText = "$" + data.Price
						}
					}
				}
			}
			if priceText == "" {
				s.Find("span").Each(func(_ int, sp *goquery.Selection) {
					t := strings.TrimSpace(sp.Text())
					if strings.HasPrefix(t, "$") || strings.HasPrefix(t, "US$") || strings.HasPrefix(t, "MXN") {
						priceText = t
					}
				})
			}
			if priceText == "" {
				s.Find("div, span").Each(func(_ int, el *goquery.Selection) {
					if priceText != "" {
						return
					}
					t := strings.TrimSpace(el.Text())
					if matched, _ := regexp.MatchString(`^\$[\d,]+\.?\d*`, t); matched {
						priceText = t
					}
				})
			}
			if priceText != "" {
				priceStr := strings.ReplaceAll(priceText, "$", "")
				priceStr = strings.ReplaceAll(priceStr, ",", "")
				priceStr = strings.TrimSpace(priceStr)
				if match := priceRe.FindString(priceStr); match != "" {
					if p, err := strconv.ParseFloat(match, 64); err == nil {
						r.Price = &p
					}
				}
			}

			r.Currency = detectCurrency(rawURL)

			if r.Title != "" && !seen[r.URL] {
				seen[r.URL] = true
				pageResults = append(pageResults, r)
			}
		})

		if len(pageResults) == 0 {
			break
		}
		allResults = append(allResults, pageResults...)

		next := doc.Find(".a-pagination .a-last:not(.a-disabled) a")
		if next.Length() == 0 {
			// Try finding any link with "Next" text
			next = doc.Find("a:contains('Next'), a:contains('next'), a:contains('→')").First()
			if next.Length() == 0 {
				break
			}
		}
		page++
	}

	return allResults, nil
}

// firstImageURL extracts the first URL from Amazon's data-a-dynamic-image JSON.
func firstImageURL(jsonStr string) string {
	re := regexp.MustCompile(`"(https?://[^"]+)"`)
	if m := re.FindStringSubmatch(jsonStr); len(m) > 1 {
		return m[1]
	}
	return ""
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
