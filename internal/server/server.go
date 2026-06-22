package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/CaptDany/kestrel/internal/handler"
	"github.com/CaptDany/kestrel/internal/notifier"
	"github.com/CaptDany/kestrel/internal/tracker"
)

type Server struct {
	Router chi.Router
	Addr   string
	Tpl    *template.Template
	Handler *handler.Handler
}

func fmtSize(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%dKB", b/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
}

func fmtDur(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(lrw, r)
		path := r.URL.Path
		if q := r.URL.RawQuery; q != "" {
			path += "?" + q
		}
		if len(path) > 55 {
			path = path[:52] + "..."
		}
		log.Printf("%s  %-6s  %-55s  %3d  %6s  %6s",
			start.Format("2006-01-02 15:04"), r.Method, path,
			lrw.Status(), fmtSize(lrw.BytesWritten()), fmtDur(time.Since(start)),
		)
	})
}

func New(addr string, tpl *template.Template, staticFS http.FileSystem) *Server {
	s := &Server{Addr: addr, Tpl: tpl}
	s.Router = chi.NewRouter()
	s.Router.Use(requestLogger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				r.RemoteAddr = strings.TrimSpace(parts[0])
			} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
				r.RemoteAddr = xri
			}
			next.ServeHTTP(w, r)
		})
	})

	s.Handler = handler.NewHandler(s.Tpl)

	s.Router.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/static")
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

	fsrv := http.FileServer(staticFS)
	s.Router.Get("/manifest.json", fsrv.ServeHTTP)
	s.Router.Get("/sw.js", fsrv.ServeHTTP)

	uploadsDir := "data/uploads"
	_ = os.MkdirAll(uploadsDir, 0755)
	s.Router.Get("/uploads/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/uploads", http.FileServer(http.Dir(uploadsDir))).ServeHTTP(w, r)
	})

	s.Router.Get("/", s.Handler.Dashboard)
	s.Router.Get("/items", s.Handler.ItemsPage)
	s.Router.Get("/items/new", s.Handler.ItemNew)
	s.Router.Get("/items/{id}/edit", s.Handler.ItemEdit)
	s.Router.Get("/schedule", s.Handler.SchedulePage)
	s.Router.Get("/history", s.Handler.HistoryPage)
	s.Router.Get("/settings", s.Handler.SettingsPage)

	s.Router.Post("/api/items", s.Handler.CreateItem)
	s.Router.Put("/api/items/{id}", s.Handler.UpdateItem)
	s.Router.Delete("/api/items/{id}", s.Handler.DeleteItem)
	s.Router.Post("/api/items/{id}/purchase", s.Handler.PurchaseItem)
	s.Router.Post("/api/items/{id}/image", s.Handler.UploadItemImage)
	s.Router.Delete("/api/items/{id}/image", s.Handler.DeleteItemImage)
	s.Router.Post("/api/scrape", s.Handler.ScrapeURL)
	s.Router.Post("/api/import/wishlist", s.Handler.ImportWishlist)
	s.Router.Post("/api/items/bulk", s.Handler.BulkCreateItems)
	s.Router.Post("/api/plan/generate", s.Handler.GeneratePlan)
	s.Router.Get("/api/plan", s.Handler.GetPlan)
	s.Router.Put("/api/settings", s.Handler.UpdateSettings)
	s.Router.Post("/api/paydays", s.Handler.CreatePayday)
	s.Router.Put("/api/paydays/{id}", s.Handler.UpdatePayday)
	s.Router.Delete("/api/paydays/{id}", s.Handler.DeletePayday)
	s.Router.Get("/api/budget-entries", s.Handler.GetBudgetEntries)
	s.Router.Post("/api/budget-entries", s.Handler.CreateBudgetEntry)
	s.Router.Delete("/api/budget-entries/{id}", s.Handler.DeleteBudgetEntry)
	s.Router.Post("/api/notify/test", s.Handler.SendTestNotification)
	s.Router.Get("/api/notifications", s.Handler.GetNotifications)
	s.Router.Put("/api/notifications/read-all", s.Handler.MarkAllNotificationsRead)
	s.Router.Put("/api/notifications/{id}/read", s.Handler.MarkNotificationRead)

	return s
}

func (s *Server) SetNotifierEngine(e *notifier.Engine) {
	if s.Handler != nil {
		s.Handler.SetNotifierEngine(e)
	}
}

func (s *Server) SetTracker(t *tracker.Tracker) {
	if s.Handler != nil {
		s.Handler.SetTracker(t)
	}
}

func (s *Server) Start() error {
	log.Printf("kestrel starting on %s", s.Addr)
	return http.ListenAndServe(s.Addr, s.Router)
}
