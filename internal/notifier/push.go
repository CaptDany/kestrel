package notifier

import "log"

type PushNotifier struct {
	enabled bool
}

func NewPushNotifier() *PushNotifier {
	return &PushNotifier{}
}

func (p *PushNotifier) Name() string { return "push" }

func (p *PushNotifier) IsEnabled() bool { return p.enabled }

func (p *PushNotifier) Configure(cfg map[string]string) error {
	enabled := cfg["notify_push_enabled"] == "true"
	globalEnabled := cfg["notify_enabled"] == "true"
	p.enabled = enabled && globalEnabled
	return nil
}

func (p *PushNotifier) Send(msg Message) error {
	if !p.enabled {
		return nil
	}
	log.Printf("[push notifier] would send: type=%s title=%s price=%.2f", msg.Type, msg.ItemTitle, msg.Price)
	return nil
}
