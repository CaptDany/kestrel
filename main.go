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
	"syscall"

	"github.com/CaptDany/kestrel/internal/config"
	"github.com/CaptDany/kestrel/internal/db"
	"github.com/CaptDany/kestrel/internal/notifier"
	"github.com/CaptDany/kestrel/internal/server"
)

//go:embed ui/templates/*.html
var templateFS embed.FS

//go:embed ui/static/*
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

	srv := server.New(cfg.Addr(), tpl, http.FS(staticSub))
	srv.SetNotifierEngine(notifEngine)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go notifEngine.Start(ctx)

	log.Printf("kestrel starting on %s", cfg.Addr())
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
