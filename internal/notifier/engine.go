package notifier

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CaptDany/kestrel/internal/db"
)

type Engine struct {
	notifiers []Notifier
	mu        sync.RWMutex
	cfg       Config
	interval  time.Duration
}

func New(notifiers []Notifier) *Engine {
	return &Engine{
		notifiers: notifiers,
		interval:  15 * time.Minute,
	}
}

func (e *Engine) SetInterval(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.interval = d
}

func (e *Engine) RefreshConfig() {
	e.mu.Lock()
	defer e.mu.Unlock()

	settings, err := db.GetAllSettings()
	if err != nil {
		log.Printf("notification engine: failed to load settings: %v", err)
		return
	}

	e.cfg = Config(settings)
	for _, n := range e.notifiers {
		if err := n.Configure(settings); err != nil {
			log.Printf("notification engine: configure %s: %v", n.Name(), err)
		}
	}
}

func (e *Engine) Start(ctx context.Context) {
	e.RefreshConfig()

	if e.cfg.Get("notify_enabled", "false") != "true" {
		log.Println("notification engine: disabled, not starting")
		return
	}

	log.Println("notification engine: started")

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	e.check()

	for {
		select {
		case <-ticker.C:
			e.check()
		case <-ctx.Done():
			log.Println("notification engine: stopped")
			return
		}
	}
}

func (e *Engine) check() {
	e.mu.RLock()
	enabled := e.cfg.Get("notify_enabled", "false") == "true"
	e.mu.RUnlock()

	if !enabled {
		return
	}

	items, err := db.GetPlanItemsReady()
	if err != nil {
		log.Printf("notification engine: query ready items: %v", err)
		return
	}

	quiet := e.isQuietHours()

	for _, item := range items {
		if quiet {
			log.Printf("notification engine: quiet hours, skipping %d", item.ItemID)
			continue
		}

		e.mu.RLock()
		alreadyNotified, err := db.GetNotificationsForItem(item.ItemID, "purchase_ready", item.ScheduledDate)
		e.mu.RUnlock()
		if err != nil {
			log.Printf("notification engine: check dup: %v", err)
			continue
		}
		if len(alreadyNotified) > 0 {
			continue
		}

		price := 0.0
		if item.ItemPrice != nil {
			price = *item.ItemPrice
		}

		msg := Message{
			Type:          "purchase_ready",
			Subject:       fmt.Sprintf("Ready to buy: %s", item.ItemTitle),
			ItemID:        item.ItemID,
			ItemTitle:     item.ItemTitle,
			ItemURL:       item.ItemURL,
			Price:         price,
			ScheduledDate: item.ScheduledDate,
		}
		msg.Body = e.buildMessageBody(msg)

		for _, n := range e.notifiers {
			if !n.IsEnabled() {
				continue
			}
			if err := n.Send(msg); err != nil {
				log.Printf("notification engine: %s send failed: %v", n.Name(), err)
				db.LogNotification(&db.NotificationLog{
					ItemID:    &item.ItemID,
					Type:      "purchase_ready",
					Channel:   n.Name(),
					Subject:   msg.Subject,
					Body:      msg.Body,
					Delivered: 0,
				})
				continue
			}
			db.LogNotification(&db.NotificationLog{
				ItemID:    &item.ItemID,
				Type:      "purchase_ready",
				Channel:   n.Name(),
				Subject:   msg.Subject,
				Body:      msg.Body,
				Delivered: 1,
			})
		}
	}
}

func (e *Engine) isQuietHours() bool {
	startStr := e.cfg.Get("notify_quiet_start", "22:00")
	endStr := e.cfg.Get("notify_quiet_end", "08:00")
	now := time.Now()

	start, err1 := time.Parse("15:04", startStr)
	end, err2 := time.Parse("15:04", endStr)
	if err1 != nil || err2 != nil {
		return false
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()

	if startMinutes <= endMinutes {
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

func (e *Engine) buildMessageBody(msg Message) string {
	body := fmt.Sprintf("Your item is now ready to be purchased!\n\n")
	body += fmt.Sprintf("Item: %s\n", msg.ItemTitle)
	if msg.ItemURL != "" {
		body += fmt.Sprintf("URL: %s\n", msg.ItemURL)
	}
	if msg.Price > 0 {
		body += fmt.Sprintf("Price: %.2f\n", msg.Price)
	}
	if msg.ScheduledDate != "" {
		body += fmt.Sprintf("Scheduled Date: %s\n", msg.ScheduledDate)
	}
	body += "\n---\nSent by kestrel Purchase Planner"
	return body
}

// Notify dispatches a message to all enabled notifiers with dedup and quiet hours.
// Returns true if at least one notifier delivered the message.
func (e *Engine) Notify(msg Message) bool {
	e.mu.RLock()
	enabled := e.cfg.Get("notify_enabled", "false") == "true"
	e.mu.RUnlock()

	if !enabled {
		return false
	}

	if e.isQuietHours() {
		log.Printf("notification engine: quiet hours, skipping %s for item %d", msg.Type, msg.ItemID)
		return false
	}

	dedupKey := e.dedupKey(msg)
	if dedupKey != "" {
		e.mu.RLock()
		alreadyNotified, err := db.GetNotificationsForItem(msg.ItemID, msg.Type, dedupKey)
		e.mu.RUnlock()
		if err == nil && len(alreadyNotified) > 0 {
			log.Printf("notification engine: already notified %s for item %d, skipping", msg.Type, msg.ItemID)
			return false
		}
	}

	delivered := false
	for _, n := range e.notifiers {
		if !n.IsEnabled() {
			continue
		}
		if err := n.Send(msg); err != nil {
			log.Printf("notification engine: %s send failed for item %d: %v", n.Name(), msg.ItemID, err)
			db.LogNotification(&db.NotificationLog{
				ItemID:    &msg.ItemID,
				Type:      msg.Type,
				Channel:   n.Name(),
				Subject:   msg.Subject,
				Body:      msg.Body,
				Delivered: 0,
			})
			continue
		}
		db.LogNotification(&db.NotificationLog{
			ItemID:    &msg.ItemID,
			Type:      msg.Type,
			Channel:   n.Name(),
			Subject:   msg.Subject,
			Body:      msg.Body,
			Delivered: 1,
		})
		delivered = true
	}
	return delivered
}

func (e *Engine) dedupKey(msg Message) string {
	switch msg.Type {
	case "price_drop":
		return time.Now().Format("2006-01-02")
	case "purchase_ready":
		return msg.ScheduledDate
	default:
		return ""
	}
}

func (e *Engine) SendTest(channel string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, n := range e.notifiers {
		if channel != "" && n.Name() != channel {
			continue
		}
		if !n.IsEnabled() {
			continue
		}
		if err := n.Send(Message{
			Type:    "test",
			Subject: "kestrel: Test Notification",
			Body:    "This is a test message from kestrel. Your notification channel is working correctly.",
		}); err != nil {
			return fmt.Errorf("%s: %w", n.Name(), err)
		}
	}
	return nil
}
