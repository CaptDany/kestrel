package extractor

import (
	"net/url"
	"testing"
)

func TestAmazonParser(t *testing.T) {
	p := &amazonParser{}
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.amazon.com/dp/B0EXAMPLE", true},
		{"https://www.amazon.co.uk/dp/B0EXAMPLE", true},
		{"https://www.amazon.de/dp/B0EXAMPLE", true},
		{"https://www.google.com", false},
		{"https://www.mercadolibre.com/item/123", false},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := p.Supports(u); got != tt.expected {
			t.Errorf("amazonParser.Supports(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestMercadoLibreParser(t *testing.T) {
	p := &mercadolibreParser{}
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.mercadolibre.com.ar/item/123", true},
		{"https://www.mercadolibre.com.mx/item/123", true},
		{"https://www.mercadolivre.com.br/item/123", true},
		{"https://www.amazon.com/dp/B0EXAMPLE", false},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := p.Supports(u); got != tt.expected {
			t.Errorf("mercadolibreParser.Supports(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestAliexpressParser(t *testing.T) {
	p := &aliexpressParser{}
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.aliexpress.com/item/123", true},
		{"https://www.aliexpress.us/item/123", true},
		{"https://www.amazon.com/dp/B0EXAMPLE", false},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := p.Supports(u); got != tt.expected {
			t.Errorf("aliexpressParser.Supports(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestIkeaParser(t *testing.T) {
	p := &ikeaParser{}
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.ikea.com/us/en/p/chair-123", true},
		{"https://www.ikea.com/gb/en/p/table-456", true},
		{"https://www.amazon.com/dp/B0EXAMPLE", false},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := p.Supports(u); got != tt.expected {
			t.Errorf("ikeaParser.Supports(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestWalmartParser(t *testing.T) {
	p := &walmartParser{}
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.walmart.com/ip/product/123", true},
		{"https://www.walmart.ca/ip/product/456", true},
		{"https://www.amazon.com/dp/B0EXAMPLE", false},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := p.Supports(u); got != tt.expected {
			t.Errorf("walmartParser.Supports(%s) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestDetectCurrency(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.amazon.com/dp/B0EXAMPLE", "USD"},
		{"https://www.amazon.co.uk/dp/B0EXAMPLE", "GBP"},
		{"https://www.amazon.de/dp/B0EXAMPLE", "EUR"},
		{"https://www.amazon.com.mx/dp/B0EXAMPLE", "MXN"},
		{"https://www.amazon.ca/dp/B0EXAMPLE", "CAD"},
		{"https://www.mercadolibre.com.ar/item/123", "ARS"},
		{"https://www.mercadolibre.com.mx/item/123", "MXN"},
		{"https://www.mercadolibre.com.br/item/123", "BRL"},
	}

	for _, tt := range tests {
		if got := detectCurrency(tt.url); got != tt.expected {
			t.Errorf("detectCurrency(%s) = %s, want %s", tt.url, got, tt.expected)
		}
	}
}

func TestGetParser(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.amazon.com/dp/B0EXAMPLE", "amazon"},
		{"https://www.mercadolibre.com.ar/item/123", "mercadolibre"},
		{"https://www.aliexpress.com/item/123", "aliexpress"},
		{"https://www.ikea.com/us/en/p/chair-123", "ikea"},
		{"https://www.walmart.com/ip/product/123", "walmart"},
		{"https://www.example.com/product/123", "generic"},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		if got := GetParser(u); got.Name() != tt.expected {
			t.Errorf("GetParser(%s) = %s, want %s", tt.url, got.Name(), tt.expected)
		}
	}
}
