package notifier

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type EmailNotifier struct {
	enabled bool
	host    string
	port    string
	user    string
	pass    string
	from    string
	to      string
}

func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{}
}

func (e *EmailNotifier) Name() string { return "email" }

func (e *EmailNotifier) IsEnabled() bool { return e.enabled }

func (e *EmailNotifier) Configure(cfg map[string]string) error {
	enabled := cfg["notify_email_enabled"] == "true"
	globalEnabled := cfg["notify_enabled"] == "true"
	e.enabled = enabled && globalEnabled
	e.host = cfg["notify_smtp_host"]
	e.port = cfg["notify_smtp_port"]
	e.user = cfg["notify_smtp_user"]
	e.pass = cfg["notify_smtp_pass"]
	e.from = cfg["notify_smtp_from"]
	e.to = cfg["notify_smtp_to"]
	if e.enabled && e.host == "" {
		return fmt.Errorf("email enabled but SMTP host not configured")
	}
	return nil
}

func (e *EmailNotifier) Send(msg Message) error {
	if !e.enabled {
		return nil
	}
	if e.host == "" || e.to == "" {
		return fmt.Errorf("email notifier not fully configured")
	}

	addr := fmt.Sprintf("%s:%s", e.host, e.port)
	subject := msg.Subject
	body := msg.Body
	if subject == "" {
		subject = fmt.Sprintf("Kestrel: %s", msg.Type)
	}
	if body == "" {
		body = e.buildBody(msg)
	}

	emailBody := e.buildEmail(subject, body)

	var auth smtp.Auth
	if e.user != "" {
		auth = smtp.PlainAuth("", e.user, e.pass, e.host)
	}

	if e.port == "465" {
		return e.sendSSL(addr, auth, emailBody)
	}
	return e.sendSTARTTLS(addr, auth, emailBody)
}

func (e *EmailNotifier) sendSTARTTLS(addr string, auth smtp.Auth, body string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: e.host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(e.to); err != nil {
		return fmt.Errorf("rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	_, err = w.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return w.Close()
}

func (e *EmailNotifier) sendSSL(addr string, auth smtp.Auth, body string) error {
	tlsCfg := &tls.Config{ServerName: e.host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(e.to); err != nil {
		return fmt.Errorf("rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	_, err = w.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}

func (e *EmailNotifier) buildEmail(subject, body string) string {
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		e.from, e.to, subject)
	return headers + body
}

func (e *EmailNotifier) buildBody(msg Message) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Item: %s\n", msg.ItemTitle))
	if msg.ItemURL != "" {
		b.WriteString(fmt.Sprintf("URL: %s\n", msg.ItemURL))
	}
	if msg.Price > 0 {
		b.WriteString(fmt.Sprintf("Price: %.2f\n", msg.Price))
	}
	if msg.ScheduledDate != "" {
		b.WriteString(fmt.Sprintf("Scheduled: %s\n", msg.ScheduledDate))
	}
	b.WriteString(fmt.Sprintf("\n---\nSent by Kestrel Purchase Planner"))
	return b.String()
}

func (e *EmailNotifier) SendTest(to string) error {
	msg := Message{
		Type:    "test",
		Subject: "Kestrel: Test Notification",
		Body:    "This is a test notification from Kestrel. If you received this, your email configuration is working.",
	}
	oldTo := e.to
	e.to = to
	err := e.Send(msg)
	e.to = oldTo
	if err != nil {
		log.Printf("test email failed: %v", err)
	}
	return err
}
