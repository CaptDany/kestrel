package notifier

import (
	"github.com/CaptDany/kestrel/internal/db"
)

type InAppNotifier struct{}

func NewInAppNotifier() *InAppNotifier {
	return &InAppNotifier{}
}

func (n *InAppNotifier) Name() string { return "inapp" }

func (n *InAppNotifier) IsEnabled() bool { return true }

func (n *InAppNotifier) Configure(cfg map[string]string) error { return nil }

func (n *InAppNotifier) Send(msg Message) error {
	_, err := db.LogNotification(&db.NotificationLog{
		ItemID:    &msg.ItemID,
		Type:      msg.Type,
		Channel:   "inapp",
		Subject:   msg.Subject,
		Body:      msg.Body,
		Delivered: 1,
		IsRead:    0,
	})
	return err
}
