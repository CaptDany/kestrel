package notifier

import "fmt"

type Message struct {
	Type          string
	Subject       string
	Body          string
	ItemID        int64
	ItemTitle     string
	ItemURL       string
	Price         float64
	ScheduledDate string
}

type Notifier interface {
	Name() string
	Send(msg Message) error
	IsEnabled() bool
	Configure(cfg map[string]string) error
}

type Config map[string]string

func (c Config) Get(key, fallback string) string {
	if v, ok := c[key]; ok && v != "" {
		return v
	}
	return fallback
}

func (c Config) GetInt(key string, fallback int) int {
	if v, ok := c[key]; ok && v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}
