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
	body += "\n---\nSent by Kestrel Purchase Planner"
	return body
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
			Subject: "Kestrel: Test Notification",
			Body:    "This is a test message from Kestrel. Your notification channel is working correctly.",
		}); err != nil {
			return fmt.Errorf("%s: %w", n.Name(), err)
		}
	}
	return nil
}
