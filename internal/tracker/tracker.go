package tracker

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"sync"
	"time"

	"github.com/CaptDany/kestrel/internal/db"
	"github.com/CaptDany/kestrel/internal/extractor"
	"github.com/CaptDany/kestrel/internal/notifier"
)

type Tracker struct {
	engine          *notifier.Engine
	mu              sync.RWMutex
	interval        time.Duration
	thresholdPct    float64
	thresholdAbs    float64
	enabled         bool
	lastCheck       time.Time
}

func New(engine *notifier.Engine) *Tracker {
	return &Tracker{
		engine:   engine,
		interval: 1 * time.Hour,
	}
}

func (t *Tracker) RefreshConfig() {
	t.mu.Lock()
	defer t.mu.Unlock()

	settings, err := db.GetAllSettings()
	if err != nil {
		log.Printf("tracker: failed to load settings: %v", err)
		return
	}

	t.enabled = settings["tracker_enabled"] == "true"

	if sec := settings["tracker_interval_seconds"]; sec != "" {
		var secs float64
		if _, err := fmt.Sscanf(sec, "%f", &secs); err == nil && secs > 0 {
			t.interval = time.Duration(secs) * time.Second
		}
	}
	if pct := settings["tracker_drop_threshold_pct"]; pct != "" {
		fmt.Sscanf(pct, "%f", &t.thresholdPct)
	}
	if abs := settings["tracker_drop_threshold_abs"]; abs != "" {
		fmt.Sscanf(abs, "%f", &t.thresholdAbs)
	}

	if t.interval < 30*time.Second {
		t.interval = 30 * time.Second
	}
}

func (t *Tracker) Start(ctx context.Context) {
	t.RefreshConfig()

	if !t.enabled {
		log.Println("tracker: disabled, not starting")
		return
	}

	log.Printf("tracker: started (interval=%s, drop threshold=%.0f%% / $%.2f)", t.interval, t.thresholdPct, t.thresholdAbs)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	t.check()

	for {
		select {
		case <-ticker.C:
			t.check()
		case <-ctx.Done():
			log.Println("tracker: stopped")
			return
		}
	}
}

func (t *Tracker) check() {
	t.mu.RLock()
	if !t.enabled {
		t.mu.RUnlock()
		return
	}
	interval := t.interval
	t.mu.RUnlock()

	// Rate-limit: don't run if last check was too recent
	t.mu.Lock()
	if time.Since(t.lastCheck) < interval/2 {
		t.mu.Unlock()
		return
	}
	t.lastCheck = time.Now()
	t.mu.Unlock()

	items, err := db.GetTrackableItems()
	if err != nil {
		log.Printf("tracker: get items: %v", err)
		return
	}

	if len(items) == 0 {
		return
	}

	log.Printf("tracker: checking %d items for price changes", len(items))

	for _, item := range items {
		t.checkItem(item)
	}
}

func (t *Tracker) checkItem(item db.Item) {
	u, err := url.Parse(item.URL)
	if err != nil {
		log.Printf("tracker: invalid url for item %d: %v", item.ID, err)
		return
	}

	parser := extractor.GetParser(u)
	result, err := parser.Extract(item.URL)
	if err != nil {
		log.Printf("tracker: extract item %d (%s): %v", item.ID, item.Title, err)
		return
	}

	if result.Price == nil {
		return
	}

	recordPrice(item.ID, result.Price, result.Currency)

	low := lowestPrice(item.ID)
	if low == nil {
		return
	}

	if *result.Price >= *low {
		return
	}

	dropAmount := *low - *result.Price
	dropPct := (dropAmount / *low) * 100

	t.mu.RLock()
	thresholdPct := t.thresholdPct
	thresholdAbs := t.thresholdAbs
	t.mu.RUnlock()

	if dropPct < thresholdPct {
		return
	}
	if thresholdAbs > 0 && dropAmount < thresholdAbs {
		return
	}

	subject := fmt.Sprintf("Price dropped on: %s", item.Title)
	body := fmt.Sprintf("Price dropped on one of your tracked items!\n\n")
	body += fmt.Sprintf("Item: %s\n", item.Title)
	body += fmt.Sprintf("URL: %s\n", item.URL)
	body += fmt.Sprintf("Previous low: $%.2f\n", *low)
	body += fmt.Sprintf("New price: $%.2f\n", *result.Price)
	body += fmt.Sprintf("Drop: $%.2f (%.1f%%)\n", dropAmount, dropPct)
	body += fmt.Sprintf("\n---\nSent by kestrel Purchase Planner")

	dropPctRounded := math.Round(dropPct*10) / 10

	t.engine.Notify(notifier.Message{
		Type:      "price_drop",
		Subject:   subject,
		Body:      body,
		ItemID:    item.ID,
		ItemTitle: item.Title,
		ItemURL:   item.URL,
		Price:     *result.Price,
	})

	log.Printf("tracker: price drop for item %d (%s): $%.2f -> $%.2f (%.1f%%)", item.ID, item.Title, *low, *result.Price, dropPctRounded)
}

func (t *Tracker) Status() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return map[string]interface{}{
		"enabled":        t.enabled,
		"interval":       t.interval.String(),
		"threshold_pct":  t.thresholdPct,
		"threshold_abs":  t.thresholdAbs,
		"last_check":     t.lastCheck.Format(time.RFC3339),
	}
}
