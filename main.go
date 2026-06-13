package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/CaptDany/kestrel/internal/config"
	"github.com/CaptDany/kestrel/internal/db"
	"github.com/CaptDany/kestrel/internal/notifier"
	"github.com/CaptDany/kestrel/internal/server"
	"github.com/CaptDany/kestrel/internal/tracker"
)

//go:embed ui/templates/*.html
var templateFS embed.FS

//go:embed ui/static
var staticFS embed.FS

func main() {
	cfg := config.Load()

	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("create db directory: %v", err)
	}

	if err := db.Init(cfg.DBPath); err != nil {
		log.Fatalf("database init: %v", err)
	}
	defer db.Close()

	funcs := template.FuncMap{
		"deref": func(f *float64) float64 {
			if f == nil {
				return 0
			}
			return *f
		},
		"derefs": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"derefi": func(i *int64) int64 {
			if i == nil {
				return 0
			}
			return *i
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"labelize": func(s string) string {
			if s == "" {
				return ""
			}
			words := strings.Split(s, "_")
			for i, w := range words {
				if len(w) > 0 {
					words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
				}
			}
			return strings.Join(words, " ")
		},
	}

	tpl := template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "ui/templates/*.html"))

	staticSub, err := fs.Sub(staticFS, "ui/static")
	if err != nil {
		log.Fatalf("static sub: %v", err)
	}

	notifEngine := notifier.New([]notifier.Notifier{
		notifier.NewEmailNotifier(),
		notifier.NewPushNotifier(),
	})

	priceTracker := tracker.New(notifEngine)

	srv := server.New(cfg.Addr(), tpl, http.FS(staticSub))
	srv.SetNotifierEngine(notifEngine)
	srv.SetTracker(priceTracker)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go notifEngine.Start(ctx)
	go priceTracker.Start(ctx)

	log.Printf("kestrel starting on %s", cfg.Addr())
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
